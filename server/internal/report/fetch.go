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
