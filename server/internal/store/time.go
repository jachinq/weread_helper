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

func YearFromPayload(payload string, fallback time.Time) int {
	m := conv.ParseJSONMap(payload)
	if ts, ok := conv.AsInt64(m["baseTime"]); ok && ts > 0 {
		if ts > 1e12 {
			ts = ts / 1000
		}
		y := time.Unix(ts, 0).In(Shanghai()).Year()
		if y >= 2010 && y <= 2100 {
			return y
		}
	}
	y := CalendarYear(fallback)
	if y < 2010 {
		return CalendarYear(time.Now())
	}
	return y
}
