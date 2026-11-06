package sse

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"tg-sender/internal/middleware"

	"github.com/google/uuid"
	"tg-sender/internal/config"
)

func newTestBroker() *Broker {
	return NewBroker(&config.Config{SSEChannelBuffer: 16})
}

func sub(t *testing.T, b *Broker, topic string) (uuid.UUID, chan Event) {
	t.Helper()
	id, ch := b.Subscribe(topic)
	t.Cleanup(func() { b.Unsubscribe(topic, id) })
	return id, ch
}

func recvWithTimeout(t *testing.T, ch chan Event) Event {
	t.Helper()
	select {
	case ev, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func assertNoEvent(t *testing.T, ch chan Event, within time.Duration) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected extra event: %+v", ev)
	case <-time.After(within):
	}
}

func TestBroadcastToTenant_EventsType_SendsSingleWrappedEvent(t *testing.T) {
	b := newTestBroker()
	defer b.Close()

	tenantID := uuid.New()
	topic := "tenant:" + tenantID.String() + ":events"
	_, ch := sub(t, b, topic)

	b.BroadcastToTenant(tenantID, "events", map[string]any{"progress": 42})

	// 修复点：events 类型只应收到 1 份包装事件（此前会额外收到裸 payload）
	ev := recvWithTimeout(t, ch)
	wrapped, ok := ev.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected wrapped map payload, got %T", ev.Payload)
	}
	if wrapped["type"] != "events" {
		t.Errorf("wrapped type = %v, want events", wrapped["type"])
	}
	inner, ok := wrapped["payload"].(map[string]any)
	if !ok || inner["progress"] != 42 {
		t.Errorf("inner payload = %v, want {progress:42}", wrapped["payload"])
	}

	assertNoEvent(t, ch, 50*time.Millisecond)
}

func TestBroadcastToTenant_OtherType_SendsSpecificAndUnified(t *testing.T) {
	b := newTestBroker()
	defer b.Close()

	tenantID := uuid.New()
	_, chSpecific := sub(t, b, "tenant:"+tenantID.String()+":new_message")
	_, chUnified := sub(t, b, "tenant:"+tenantID.String()+":events")

	b.BroadcastToTenant(tenantID, "new_message", map[string]any{"text": "hi"})

	ev1 := recvWithTimeout(t, chSpecific)
	if payload := ev1.Payload.(map[string]any); payload["text"] != "hi" {
		t.Errorf("specific topic payload = %v", ev1.Payload)
	}
	ev2 := recvWithTimeout(t, chUnified)
	if wrapped := ev2.Payload.(map[string]any); wrapped["type"] != "new_message" {
		t.Errorf("unified wrapped type = %v", wrapped["type"])
	}
}

func TestBroadcastGlobal_ReachesGlobalTopicNotTenantTopics(t *testing.T) {
	b := newTestBroker()
	defer b.Close()

	_, chGlobal := sub(t, b, globalTopic)
	_, chOther := sub(t, b, "tenant:"+uuid.New().String()+":events")

	b.BroadcastGlobal("memory_high", map[string]any{"value": 900})

	ev := recvWithTimeout(t, chGlobal)
	if wrapped := ev.Payload.(map[string]any); wrapped["type"] != "memory_high" {
		t.Errorf("global wrapped type = %v, want memory_high", wrapped["type"])
	}

	assertNoEvent(t, chOther, 50*time.Millisecond)
}


func TestHandlerUsesLogicalSSEEventName(t *testing.T) {
	b := newTestBroker()
	defer b.Close()

	tenantID := uuid.New()
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(middleware.TenantContextKey, tenantID.String())
			return next(c)
		}
	})
	e.GET("/events", b.Handler("events"))
	server := httptest.NewServer(e)
	defer server.Close()

	resp, err := http.Get(server.URL + "/events")
	if err != nil { t.Fatalf("open SSE stream: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { t.Fatalf("status = %d, want 200", resp.StatusCode) }

	scanner := bufio.NewScanner(resp.Body)
	deadline := time.After(time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for connected frame")
		default:
		}
		if !scanner.Scan() { t.Fatalf("SSE stream ended: %v", scanner.Err()) }
		if scanner.Text() == "event: connected" { break }
	}
	b.BroadcastToTenant(tenantID, "events", map[string]any{"task_id": "task-1"})
	foundLogicalEvent := false
	for !foundLogicalEvent {
		if !scanner.Scan() { t.Fatalf("SSE stream ended before event: %v", scanner.Err()) }
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			if line == "event: events" { foundLogicalEvent = true }
			if strings.HasPrefix(line, "event: tenant:") { t.Fatalf("internal tenant topic leaked: %s", line) }
		}
	}
	if !foundLogicalEvent { t.Fatal(fmt.Sprintf("logical SSE event not found")) }
}
