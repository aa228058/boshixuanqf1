package sse

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/config"
	"tg-sender/internal/middleware"
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

func TestHandlerEmitsNamedHeartbeatEvent(t *testing.T) {
	b := NewBroker(&config.Config{SSEChannelBuffer: 16, SSEHeartbeat: 10 * time.Millisecond})
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
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer resp.Body.Close()

	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("SSE stream ended before heartbeat")
			}
			if line == "event: heartbeat" {
				return
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for named heartbeat event")
		}
	}
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
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	readLine := func() string {
		t.Helper()
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("SSE stream ended unexpectedly")
			}
			return line
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for SSE frame")
			return ""
		}
	}
	for readLine() != "event: connected" {
	}
	b.BroadcastToTenant(tenantID, "events", map[string]any{"task_id": "task-1"})
	for {
		line := readLine()
		if strings.HasPrefix(line, "event: tenant:") {
			t.Fatalf("internal tenant topic leaked: %s", line)
		}
		if line == "event: events" {
			break
		}
	}
}


func TestBroadcastQueueOverflowEnqueuesResyncMarker(t *testing.T) {
	b := NewBroker(&config.Config{SSEChannelBuffer: 2})
	defer b.Close()

	_, ch := sub(t, b, "tenant:test:events")
	b.Broadcast("tenant:test:events", map[string]any{"seq": 1})
	b.Broadcast("tenant:test:events", map[string]any{"seq": 2})
	b.Broadcast("tenant:test:events", map[string]any{"seq": 3})

	first := recvWithTimeout(t, ch)
	if marker, ok := first.Payload.(map[string]any); !ok || marker["type"] != "resync_required" {
		t.Fatalf("first queued event should be resync marker after oldest event eviction, got %#v", first.Payload)
	}
	if len(ch) != 0 {
		t.Fatalf("expected only one resync marker, queue length = %d", len(ch))
	}
}
