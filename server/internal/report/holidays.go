package report

import "time"

type holidayDef struct {
	Name string
	Month time.Month
	Day  int
}

var fixedHolidays = []holidayDef{
	{Name: "元旦", Month: time.January, Day: 1},
	{Name: "世界读书日", Month: time.April, Day: 23},
	{Name: "劳动节", Month: time.May, Day: 1},
	{Name: "国庆节", Month: time.October, Day: 1},
}

// 农历正月初一（公历），2018–2030。
var springFestival = map[int][2]int{
	2018: {2, 16},
	2019: {2, 5},
	2020: {1, 25},
	2021: {2, 12},
	2022: {2, 1},
	2023: {1, 22},
	2024: {2, 10},
	2025: {1, 29},
	2026: {2, 17},
	2027: {2, 6},
	2028: {1, 26},
	2029: {2, 13},
	2030: {2, 3},
}

func holidaysForYear(year int) []holiday {
	loc := shanghai()
	out := make([]holiday, 0, len(fixedHolidays)+1)
	if md, ok := springFestival[year]; ok {
		out = append(out, holiday{
			Name: "春节",
			Date: time.Date(year, time.Month(md[0]), md[1], 0, 0, 0, 0, loc),
		})
	}
	for _, h := range fixedHolidays {
		out = append(out, holiday{
			Name: h.Name,
			Date: time.Date(year, h.Month, h.Day, 0, 0, 0, 0, loc),
		})
	}
	return out
}

type holiday struct {
	Name string
	Date time.Time
}

func shanghai() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}
