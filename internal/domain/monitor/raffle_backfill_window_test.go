package monitor

import (
	"testing"
	"time"
)

func TestRaffleBackfillWindowFirstRunUsesDefault(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if got := RaffleBackfillWindow(time.Time{}, now); got != RaffleBackfillDefaultWindow {
		t.Fatalf("首次补拉窗口 = %v, want %v", got, RaffleBackfillDefaultWindow)
	}
}

func TestRaffleBackfillWindowCoversTheGap(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// 上次补拉是 40 分钟前 → 只补这 40 分钟
	last := now.Add(-40 * time.Minute)
	if got := RaffleBackfillWindow(last, now); got != 40*time.Minute {
		t.Fatalf("窗口 = %v, want 40m", got)
	}
}

func TestRaffleBackfillWindowClampsBothEnds(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// 刚补过 10 秒 → 抬到下限，避免为几十秒扫 50 个会话
	if got := RaffleBackfillWindow(now.Add(-10*time.Second), now); got != RaffleBackfillMinWindow {
		t.Fatalf("窗口 = %v, want %v", got, RaffleBackfillMinWindow)
	}
	// 停了两天 → 压到默认上限 12 小时
	if got := RaffleBackfillWindow(now.Add(-48*time.Hour), now); got != RaffleBackfillDefaultWindow {
		t.Fatalf("窗口 = %v, want %v", got, RaffleBackfillDefaultWindow)
	}
}

// 时间倒退（时钟回拨）不能让窗口变成负值。
func TestRaffleBackfillWindowHandlesClockSkew(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if got := RaffleBackfillWindow(now.Add(time.Hour), now); got != RaffleBackfillMinWindow {
		t.Fatalf("时钟回拨时窗口 = %v, want %v", got, RaffleBackfillMinWindow)
	}
}
