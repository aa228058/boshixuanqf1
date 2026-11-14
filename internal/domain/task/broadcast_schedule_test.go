package task

import (
	"testing"
	"time"
)

func TestNextBroadcastRunDailyUsesConfiguredTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 11, 12, 8, 0, 0, 0, loc)
	got, err := NextBroadcastRun(map[string]any{
		"schedule_mode": "daily", "timezone": "America/Los_Angeles", "local_time": "09:30",
	}, after, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 12, 9, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNextBroadcastRunWeeklyUsesISOWeekdays(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	// Thursday, ISO weekday 4; only schedule Friday (5) and Monday (1).
	after := time.Date(2026, 11, 12, 10, 0, 0, 0, loc)
	got, err := NextBroadcastRun(map[string]any{
		"schedule_mode": "weekly", "timezone": "Asia/Shanghai",
		"local_time": "09:00", "weekdays": []any{float64(1), float64(5)},
	}, after, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 13, 9, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNextBroadcastRunDailySkipsDSTGap(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	// 02:30 does not exist on the 2026 spring-forward date.
	after := time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	got, err := NextBroadcastRun(map[string]any{
		"schedule_mode": "daily", "timezone": "America/Los_Angeles", "local_time": "02:30",
	}, after, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 3, 9, 2, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNextBroadcastRunKeepsLegacyIntervalMode(t *testing.T) {
	after := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	got, err := NextBroadcastRun(map[string]any{}, after, 90)
	if err != nil {
		t.Fatal(err)
	}
	if want := after.Add(90 * time.Minute); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNextBroadcastRunDailyDoesNotRepeatDuringDSTFold(t *testing.T) {
	// After the first 01:30 PDT occurrence, skip the repeated 01:30 PST
	// occurrence so a daily task cannot run twice on the same local date.
	after := time.Date(2026, 11, 1, 8, 45, 0, 0, time.UTC)
	got, err := NextBroadcastRun(map[string]any{
		"schedule_mode": "daily", "timezone": "America/Los_Angeles", "local_time": "01:30",
	}, after, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s, want next day's schedule %s", got, want)
	}
}

func TestNextBroadcastRunDailyUsesFirstDSTFoldOccurrenceWhenBothAreFuture(t *testing.T) {
	// 2026-11-01 00:45 PDT; the first 01:30 occurrence is at 08:30 UTC.
	after := time.Date(2026, 11, 1, 7, 45, 0, 0, time.UTC)
	got, err := NextBroadcastRun(map[string]any{
		"schedule_mode": "daily", "timezone": "America/Los_Angeles", "local_time": "01:30",
	}, after, 60)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s, want first DST-fold occurrence %s", got, want)
	}
}

func TestParseLocalDateTimeRejectsDSTGap(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseLocalDateTime("2026-03-08T02:30", loc); err == nil {
		t.Fatal("expected nonexistent spring-forward local time to be rejected")
	}
}

func TestParseLocalDateTimeUsesFirstDSTFoldOccurrence(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseLocalDateTime("2026-11-01T01:30", loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s, want first DST-fold occurrence %s", got, want)
	}
}

func TestNextIntervalRunAlignsToPlan(t *testing.T) {
	planned := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	// 10:50 跑完、间隔 60 分钟 → 下一轮仍是 11:00（对齐计划，不累积漂移）
	now := planned.Add(50 * time.Minute)
	if got := NextIntervalRun(&planned, now, 60); !got.Equal(planned.Add(60 * time.Minute)) {
		t.Fatalf("got %v, want 11:00", got)
	}
}

func TestNextIntervalRunSkipsCatchUpBeyondOneCycle(t *testing.T) {
	planned := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	// 11:30 才跑完：对齐点 11:00 已过去一整轮，跳到 now+60m，而不是立刻补跑
	now := planned.Add(90 * time.Minute)
	want := now.Add(60 * time.Minute)
	if got := NextIntervalRun(&planned, now, 60); !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNextIntervalRunWithoutPlanUsesNow(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	if got := NextIntervalRun(nil, now, 30); !got.Equal(now.Add(30 * time.Minute)) {
		t.Fatalf("got %v, want now+30m", got)
	}
	var zero time.Time
	if got := NextIntervalRun(&zero, now, 30); !got.Equal(now.Add(30 * time.Minute)) {
		t.Fatalf("零值计划时间应退回 now+interval, got %v", got)
	}
}

func TestNextLoopRunKeepsCalendarMode(t *testing.T) {
	payload := map[string]any{"schedule_mode": "daily", "timezone": "Asia/Shanghai", "local_time": "09:00"}
	planned := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)
	got, err := NextLoopRun(payload, now, &planned, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.After(now) {
		t.Fatalf("日历模式应给出未来时间, got %v", got)
	}
}
