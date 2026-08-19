package store

import (
	"time"

	"github.com/jachin/weread-helper/internal/conv"
)

func Shanghai() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func CalendarYear(t time.Time) int {
	return t.In(Shanghai()).Year()
}

func YearBounds(year int) (fromTs, toTs int64) {
	loc := Shanghai()
	from := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	to := time.Date(year+1, 1, 1, 0, 0, 0, 0, loc)
	return from.Unix(), to.Unix()
}

func AnnualBaseTime(year int) int64 {
	loc := Shanghai()
	return time.Date(year, 1, 1, 0, 0, 0, 0, loc).Unix()
}

func payloadTime(payload string) (time.Time, bool) {
	m := conv.ParseJSONMap(payload)
	ts, ok := conv.AsInt64(m["baseTime"])
	if !ok || ts <= 0 {
		return time.Time{}, false
	}
	if ts > 1e12 {
		ts = ts / 1000
	}
	t := time.Unix(ts, 0).In(Shanghai())
	if t.Year() < 2010 || t.Year() > 2100 {
		return time.Time{}, false
	}
	return t, true
}

func YearFromPayload(payload string, fallback time.Time) int {
	if t, ok := payloadTime(payload); ok {
		return t.Year()
	}
	y := CalendarYear(fallback)
	if y < 2010 {
		return CalendarYear(time.Now())
	}
	return y
}

func MonthKey(t time.Time) int64 {
	t = t.In(Shanghai())
	return int64(t.Year()*100 + int(t.Month()))
}

func MonthKeyFromYMD(year, month int) int64 {
	if month < 1 || month > 12 || year < 2010 {
		return 0
	}
	return int64(year*100 + month)
}

func WeekMonday(t time.Time) time.Time {
	t = t.In(Shanghai())
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, Shanghai())
	off := int(day.Weekday() - time.Monday)
	if off < 0 {
		off += 7
	}
	return day.AddDate(0, 0, -off)
}

func WeekKey(t time.Time) int64 {
	m := WeekMonday(t)
	return int64(m.Year()*10000 + int(m.Month())*100 + m.Day())
}

func TimeFromMonthKey(key int64) time.Time {
	y := int(key / 100)
	m := int(key % 100)
	if m < 1 || m > 12 {
		return time.Time{}
	}
	return time.Date(y, time.Month(m), 1, 0, 0, 0, 0, Shanghai())
}

func TimeFromWeekKey(key int64) time.Time {
	y := int(key / 10000)
	rest := int(key % 10000)
	m := rest / 100
	d := rest % 100
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return time.Time{}
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, Shanghai())
	if t.Month() != time.Month(m) || t.Day() != d {
		return time.Time{}
	}
	return WeekMonday(t)
}

func PeriodKeyFromPayload(mode, payload string, now time.Time) int64 {
	switch mode {
	case "annually":
		return int64(YearFromPayload(payload, now))
	case "monthly":
		if t, ok := payloadTime(payload); ok {
			return MonthKey(t)
		}
		return MonthKey(now)
	case "weekly":
		if t, ok := payloadTime(payload); ok {
			return WeekKey(t)
		}
		return WeekKey(now)
	default:
		return 0
	}
}
