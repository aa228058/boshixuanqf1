package grouplib

import (
	"testing"
	"time"
)

func TestTriggerMergeIsNonBlocking(t *testing.T) {
	e := NewEngine(nil, nil, nil)
	// 通道容量 1：连续触发不能阻塞调用方（落库 goroutine 里调用）
	for i := 0; i < 5; i++ {
		done := make(chan struct{})
		go func() { e.TriggerMerge(); close(done) }()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("TriggerMerge 阻塞了调用方")
		}
	}
}

func TestMergeKickDebounce(t *testing.T) {
	e := NewEngine(nil, nil, nil)
	if !e.mergeKickAllowed() {
		t.Fatal("从未跑过时应允许立即聚合")
	}
	e.lastRun.Store(time.Now().UnixNano())
	if e.mergeKickAllowed() {
		t.Fatal("刚跑过一轮，18 秒内的唤醒应被去抖拦截")
	}
	e.lastRun.Store(time.Now().Add(-mergeKickMinGap - time.Second).UnixNano())
	if !e.mergeKickAllowed() {
		t.Fatal("超过最小间隔后应允许聚合")
	}
}
