package report

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jachin/weread-helper/internal/store"
	"github.com/jachin/weread-helper/internal/weread"
)

func PullAnnual(client *weread.Client, st *store.Store, year int) error {
	var lastYear int
	for _, bt := range annualBaseTimes(year) {
		data, err := client.ReadStats("annually", bt)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		got := store.YearFromPayload(string(raw), time.Time{})
		lastYear = got
		if got == year {
			return st.PutStatsYear("annually", int64(year), string(raw))
		}
	}
	if lastYear == 0 {
		return fmt.Errorf("官方未返回 %d 年阅读统计", year)
	}
	return fmt.Errorf("官方返回的是 %d 年数据，不是 %d 年", lastYear, year)
}

func PullMonthly(client *weread.Client, st *store.Store, monthKey int64) error {
	start := store.TimeFromMonthKey(monthKey)
	if start.IsZero() {
		return fmt.Errorf("月份无效")
	}
	var last int64
	for _, bt := range monthlyBaseTimes(start) {
		data, err := client.ReadStats("monthly", bt)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		got := store.PeriodKeyFromPayload("monthly", string(raw), time.Time{})
		last = got
		if got == monthKey {
			return st.PutStatsYear("monthly", monthKey, string(raw))
		}
	}
	if last == 0 {
		return fmt.Errorf("官方未返回 %s 阅读统计", start.Format("2006年1月"))
	}
	gotT := store.TimeFromMonthKey(last)
	return fmt.Errorf("官方返回的是 %s 数据，不是 %s", gotT.Format("2006年1月"), start.Format("2006年1月"))
}

func PullWeekly(client *weread.Client, st *store.Store, weekKey int64) error {
	mon := store.TimeFromWeekKey(weekKey)
	if mon.IsZero() {
		return fmt.Errorf("周次无效")
	}
	want := store.WeekKey(mon)
	var last int64
	for _, bt := range weeklyBaseTimes(mon) {
		data, err := client.ReadStats("weekly", bt)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		got := store.PeriodKeyFromPayload("weekly", string(raw), time.Time{})
		last = got
		if got == want {
			return st.PutStatsYear("weekly", want, string(raw))
		}
	}
	if last == 0 {
		return fmt.Errorf("官方未返回 %s 当周阅读统计", mon.Format("2006-01-02"))
	}
	gotT := store.TimeFromWeekKey(last)
	return fmt.Errorf("官方返回的是 %s 当周数据，不是 %s 当周", gotT.Format("2006-01-02"), mon.Format("2006-01-02"))
}

func annualBaseTimes(year int) []int64 {
	loc := store.Shanghai()
	seen := map[int64]bool{}
	var out []int64
	add := func(t time.Time) {
		sec := t.Unix()
		if sec <= 0 || seen[sec] {
			return
		}
		seen[sec] = true
		out = append(out, sec)
	}
	add(time.Date(year, 1, 1, 0, 0, 0, 0, loc))
	add(time.Date(year, 6, 15, 12, 0, 0, 0, loc))
	add(time.Date(year, 12, 31, 12, 0, 0, 0, loc))
	add(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
	add(time.Date(year, 12, 31, 16, 0, 0, 0, time.UTC))
	return out
}

func monthlyBaseTimes(start time.Time) []int64 {
	loc := store.Shanghai()
	y, m := start.Year(), start.Month()
	seen := map[int64]bool{}
	var out []int64
	add := func(t time.Time) {
		sec := t.Unix()
		if sec <= 0 || seen[sec] {
			return
		}
		seen[sec] = true
		out = append(out, sec)
	}
	add(time.Date(y, m, 1, 0, 0, 0, 0, loc))
	add(time.Date(y, m, 15, 12, 0, 0, 0, loc))
	add(time.Date(y, m+1, 0, 12, 0, 0, 0, loc))
	add(time.Date(y, m, 1, 0, 0, 0, 0, time.UTC))
	return out
}

func weeklyBaseTimes(monday time.Time) []int64 {
	loc := store.Shanghai()
	mon := store.WeekMonday(monday)
	y, m, d := mon.Date()
	seen := map[int64]bool{}
	var out []int64
	add := func(t time.Time) {
		sec := t.Unix()
		if sec <= 0 || seen[sec] {
			return
		}
		seen[sec] = true
		out = append(out, sec)
	}
	add(time.Date(y, m, d, 0, 0, 0, 0, loc))
	add(time.Date(y, m, d, 12, 0, 0, 0, loc))
	add(mon.AddDate(0, 0, 3).In(loc))
	add(mon.AddDate(0, 0, 6).Add(12 * time.Hour))
	add(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
	return out
}
