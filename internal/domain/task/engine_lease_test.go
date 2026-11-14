package task

import (
	"testing"
	"time"
)

func TestLeaseHeartbeatInterval(t *testing.T) {
	cases := map[int]time.Duration{
		0:   30 * time.Second, // 未配置时退化为下限
		1:   30 * time.Second,
		3:   1 * time.Minute,
		15:  5 * time.Minute,
		40:  5 * time.Minute, // 默认 40 分钟锁 → 夹到上限 5 分钟
		300: 5 * time.Minute,
	}
	for lockTimeoutMin, want := range cases {
		if got := leaseHeartbeatInterval(lockTimeoutMin); got != want {
			t.Fatalf("leaseHeartbeatInterval(%d) = %v, want %v", lockTimeoutMin, got, want)
		}
	}
}

// 心跳必须严格短于租约长度，否则续租还没发生任务就被判超时。
func TestLeaseHeartbeatShorterThanLease(t *testing.T) {
	for _, minutes := range []int{1, 3, 5, 10, 40, 120} {
		lease := time.Duration(minutes) * time.Minute
		hb := leaseHeartbeatInterval(minutes)
		if hb >= lease && lease > 90*time.Second {
			t.Fatalf("锁 %v 的心跳 %v 不短于租约", lease, hb)
		}
	}
}
