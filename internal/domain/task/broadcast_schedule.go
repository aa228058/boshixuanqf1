package task

import (
	"fmt"
	"sort"
	"strconv"
	"time"
	_ "time/tzdata"
)

// NextIntervalRun 计算"interval 模式"循环任务的下一轮时间。
//
// 原实现用「本轮完成时刻 + 间隔」，每轮都会把执行耗时累加进去，跑几十轮后
// 实际开跑时间能漂出几十分钟。这里改成对齐「上一轮计划时刻 + 间隔」。
//
// 但如果对齐后的时间已经过去超过一个完整周期（任务被暂停后恢复、队列长时间
// 拥塞、进程停机很久），说明已经落下若干轮：此时直接跳到 now+间隔，跳过补跑，
// 否则会连续多轮连着发，把目标群刷爆。
func NextIntervalRun(planned *time.Time, now time.Time, intervalMinutes int) time.Time {
	if intervalMinutes <= 0 {
		intervalMinutes = 60
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	if planned == nil || planned.IsZero() {
		return now.Add(interval)
	}
	next := planned.Add(interval)
	if !next.After(now) {
		return now.Add(interval)
	}
	return next
}

// NextLoopRun 是循环任务重排的入口：interval 模式走对齐逻辑，日历模式沿用
// 原有的按本地墙钟计算。
func NextLoopRun(payload map[string]any, now time.Time, planned *time.Time, intervalMinutes int) (time.Time, error) {
	mode, _ := payload["schedule_mode"].(string)
	if mode == "" || mode == "interval" {
		return NextIntervalRun(planned, now, intervalMinutes), nil
	}
	return NextBroadcastRun(payload, now, intervalMinutes)
}

// NextBroadcastRun calculates the next recurring run. Calendar modes use the
// tenant-selected IANA timezone and local wall-clock time; interval mode remains
// backward-compatible with existing tasks.
func NextBroadcastRun(payload map[string]any, after time.Time, intervalMinutes int) (time.Time, error) {
	mode, _ := payload["schedule_mode"].(string)
	if mode == "" || mode == "interval" {
		if intervalMinutes <= 0 {
			intervalMinutes = 60
		}
		return after.Add(time.Duration(intervalMinutes) * time.Minute), nil
	}
	if mode != "daily" && mode != "weekly" {
		return time.Time{}, fmt.Errorf("unsupported broadcast schedule_mode %q", mode)
	}

	zone, _ := payload["timezone"].(string)
	if zone == "" {
		return time.Time{}, fmt.Errorf("calendar schedule requires timezone")
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid schedule timezone %q: %w", zone, err)
	}
	localTime, _ := payload["local_time"].(string)
	clock, err := time.Parse("15:04", localTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid schedule local_time %q: %w", localTime, err)
	}
	hour, minute := clock.Hour(), clock.Minute()

	var weekdays map[time.Weekday]bool
	if mode == "weekly" {
		weekdays = make(map[time.Weekday]bool)
		switch values := payload["weekdays"].(type) {
		case []any:
			for _, value := range values {
				if day, ok := scheduleInt(value); ok && day >= 1 && day <= 7 {
					weekdays[time.Weekday(day%7)] = true // ISO 7 (Sunday) maps to time.Sunday.
				}
			}
		case []int:
			for _, day := range values {
				if day >= 1 && day <= 7 {
					weekdays[time.Weekday(day%7)] = true
				}
			}
		}
		if len(weekdays) == 0 {
			return time.Time{}, fmt.Errorf("weekly schedule requires weekdays in ISO range 1..7")
		}
	}

	localAfter := after.In(loc)
	for offset := 0; offset <= 8; offset++ {
		date := time.Date(localAfter.Year(), localAfter.Month(), localAfter.Day()+offset, 0, 0, 0, 0, loc)
		if mode == "weekly" && !weekdays[date.Weekday()] {
			continue
		}
		// Resolve every instant matching this local wall-clock time. There
		// can be zero (spring-forward gap), one (normal day), or two (fall-back).
		// This avoids relying on time.Date's implementation-defined choice in a DST fold.
		candidates := localScheduleCandidates(date, hour, minute, loc)
		if len(candidates) == 0 {
			continue // The requested wall-clock time does not exist on this date.
		}
		// For an ambiguous fall-back time, always use the first occurrence.
		// If that occurrence has passed, skip this date rather than sending twice.
		if candidates[0].After(after) {
			return candidates[0], nil
		}
	}
	return time.Time{}, fmt.Errorf("no future calendar slot found for schedule")
}

func scheduleInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), v == float64(int(v))
	case string:
		n, err := strconv.Atoi(v)
		return n, err == nil
	default:
		return 0, false
	}
}

func localScheduleCandidates(date time.Time, hour, minute int, loc *time.Location) []time.Time {
	// Treat the requested local clock as a UTC tuple, then test nearby timezone
	// offsets to discover all real instants that map back to that exact tuple.
	wallClock := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.UTC)
	offsets := make(map[int]struct{})
	for delta := -36 * time.Hour; delta <= 36*time.Hour; delta += 3 * time.Hour {
		_, offset := wallClock.Add(delta).In(loc).Zone()
		offsets[offset] = struct{}{}
	}

	candidates := make([]time.Time, 0, 2)
	for offset := range offsets {
		candidate := wallClock.Add(-time.Duration(offset) * time.Second)
		local := candidate.In(loc)
		y, m, d := local.Date()
		h, min, _ := local.Clock()
		if y == date.Year() && m == date.Month() && d == date.Day() && h == hour && min == minute {
			candidates = append(candidates, candidate)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Before(candidates[j])
	})
	return candidates
}

// ParseLocalDateTime resolves a datetime-local value in the supplied location.
// For an ambiguous fall-back time it consistently selects the first occurrence;
// nonexistent spring-forward wall times are rejected.
func ParseLocalDateTime(value string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		return time.Time{}, fmt.Errorf("timezone is required")
	}
	parsed, err := time.Parse("2006-01-02T15:04", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid local date/time %q: %w", value, err)
	}
	date := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
	candidates := localScheduleCandidates(date, parsed.Hour(), parsed.Minute(), loc)
	if len(candidates) == 0 {
		return time.Time{}, fmt.Errorf("local date/time %q does not exist in timezone %s", value, loc)
	}
	return candidates[0], nil
}
