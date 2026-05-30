package timeutil

import (
	"testing"
	"time"
)

func mustSH(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestDateString_CrossesDayBoundaryByTimezone(t *testing.T) {
	loc := mustSH(t)
	// UTC 2026-05-05T20:00 == Asia/Shanghai 2026-05-06T04:00 → 日期应归到 05-06
	utc := time.Date(2026, 5, 5, 20, 0, 0, 0, time.UTC)
	if got := DateString(utc, loc); got != "2026-05-06" {
		t.Errorf("DateString = %q, want 2026-05-06", got)
	}
}

func TestPeriodRange_7dIncludesToday(t *testing.T) {
	loc := mustSH(t)
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, loc)
	start, end := PeriodRange("7d", now, loc)
	if DateString(start, loc) != "2026-05-25" {
		t.Errorf("7d start = %s, want 2026-05-25", DateString(start, loc))
	}
	// end 是半开，end 前一天应是今天
	if DateString(end.AddDate(0, 0, -1), loc) != "2026-05-31" {
		t.Errorf("7d end-1 = %s, want 2026-05-31", DateString(end.AddDate(0, 0, -1), loc))
	}
}

func TestPeriodRange_Today(t *testing.T) {
	loc := mustSH(t)
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, loc)
	start, _ := PeriodRange("today", now, loc)
	if DateString(start, loc) != "2026-05-31" {
		t.Errorf("today start = %s, want 2026-05-31", DateString(start, loc))
	}
}

func TestCustomRange(t *testing.T) {
	loc := mustSH(t)
	start, end, err := CustomRange("2026-05-01", "2026-05-07", loc)
	if err != nil {
		t.Fatal(err)
	}
	if DateString(start, loc) != "2026-05-01" {
		t.Errorf("start = %s", DateString(start, loc))
	}
	if DateString(end.AddDate(0, 0, -1), loc) != "2026-05-07" {
		t.Errorf("end-1 = %s, want 2026-05-07", DateString(end.AddDate(0, 0, -1), loc))
	}
}
