package store

import (
	"testing"
	"time"
)

func TestAnnualBaseTime2026(t *testing.T) {
	got := AnnualBaseTime(2026)
	const want int64 = 1767196800
	if got != want {
		t.Fatalf("AnnualBaseTime(2026)=%d want %d", got, want)
	}
}

func TestMonthAndWeekKeys(t *testing.T) {
	loc := Shanghai()
	jul := time.Date(2026, 7, 15, 12, 0, 0, 0, loc)
	if got := MonthKey(jul); got != 202607 {
		t.Fatalf("MonthKey=%d want 202607", got)
	}
	mon := WeekMonday(time.Date(2026, 8, 19, 9, 0, 0, 0, loc))
	if mon.Weekday() != time.Monday {
		t.Fatalf("weekday %s", mon.Weekday())
	}
	if mon.Format("2006-01-02") != "2026-08-17" {
		t.Fatalf("monday %s", mon.Format("2006-01-02"))
	}
	if WeekKey(jul) != WeekKey(WeekMonday(jul)) {
		t.Fatalf("week key mismatch")
	}
	back := TimeFromMonthKey(202607)
	if back.Format("2006-01") != "2026-07" {
		t.Fatalf("TimeFromMonthKey %s", back)
	}
}