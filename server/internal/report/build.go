package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jachin/weread-helper/internal/conv"
	"github.com/jachin/weread-helper/internal/store"
)

type MissingSnapshotError struct {
	Year int
}

func (e *MissingSnapshotError) Error() string {
	return fmt.Sprintf("尚无 %d 年官方阅读快照", e.Year)
}

type BookCard struct {
	BookID string `json:"bookId,omitempty"`
	Title  string `json:"title"`
	Author string `json:"author,omitempty"`
	Cover  string `json:"cover,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

type Overview struct {
	TotalReadTime           int64  `json:"totalReadTime"`
	TotalReadTimeFormatted  string `json:"totalReadTimeFormatted"`
	DayAverageReadTime      int64  `json:"dayAverageReadTime"`
	DayAverageFormatted     string `json:"dayAverageReadTimeFormatted"`
	ReadDays                int64  `json:"readDays"`
	BooksRead               string `json:"booksRead,omitempty"`
	BooksFinished           string `json:"booksFinished,omitempty"`
	NoteCount               int    `json:"noteCount"`
	HighlightCount          int    `json:"highlightCount"`
	ReviewCount             int    `json:"reviewCount"`
}

type MonthBar struct {
	Month   int    `json:"month"`
	Seconds int64  `json:"seconds"`
	Label   string `json:"label"`
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type MonthBook struct {
	Month int      `json:"month"`
	Book  BookCard `json:"book"`
	Count int      `json:"count"`
}

type HolidayRead struct {
	Name      string    `json:"name"`
	Date      string    `json:"date"`
	Read      bool      `json:"read"`
	BookKnown bool      `json:"bookKnown"`
	Book      *BookCard `json:"book,omitempty"`
}

type Report struct {
	Year            int           `json:"year"`
	FetchedAt       int64         `json:"fetchedAt"`
	Overview        Overview      `json:"overview"`
	Months          []MonthBar    `json:"months"`
	PeakMonth       *MonthBar     `json:"peakMonth,omitempty"`
	Cheer           string        `json:"cheer"`
	Favorite        *BookCard     `json:"favorite,omitempty"`
	FirstRead       *BookCard     `json:"firstRead,omitempty"`
	MostHighlights  *BookCard     `json:"mostHighlights,omitempty"`
	Rarest          *BookCard     `json:"rarest,omitempty"`
	Immersed        *BookCard     `json:"immersed,omitempty"`
	LateNight       *BookCard     `json:"lateNight,omitempty"`
	Holidays        []HolidayRead `json:"holidays,omitempty"`
	MonthBooks      []MonthBook   `json:"monthBooks,omitempty"`
	Categories      []NamedCount  `json:"categories,omitempty"`
	Hours           []int64       `json:"hours,omitempty"`
	HoursUnit       string        `json:"hoursUnit,omitempty"`
	Authors         []NamedCount  `json:"authors,omitempty"`
	Copyrights      []NamedCount  `json:"copyrights,omitempty"`
	CopyrightSource string        `json:"copyrightSource,omitempty"`
}

func Build(st *store.Store, year int) (*Report, error) {
	payload, fetchedAt, err := st.GetStatsYear("annually", int64(year))
	if err != nil {
		return nil, err
	}
	if payload == "" {
		return nil, &MissingSnapshotError{Year: year}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, err
	}

	fromTs, toTs := store.YearBounds(year)
	notes, err := st.ListYearNotes(fromTs, toTs)
	if err != nil {
		return nil, err
	}

	total, _ := conv.AsInt64(data["totalReadTime"])
	avg, _ := conv.AsInt64(data["dayAverageReadTime"])
	days, _ := conv.AsInt64(data["readDays"])
	booksRead, booksFinished := parseReadStat(data["readStat"])

	hl, rv := 0, 0
	for _, n := range notes {
		if n.Kind == "highlight" {
			hl++
		} else {
			rv++
		}
	}

	months := parseMonthSeconds(data, year)
	var peak *MonthBar
	for i := range months {
		if peak == nil || months[i].Seconds > peak.Seconds {
			cp := months[i]
			peak = &cp
		}
	}
	if peak != nil && peak.Seconds <= 0 {
		peak = nil
	}

	slots := parsePreferBooks(data["preferBooks"])
	hours, hoursUnit := preferHours(data["preferTime"], notes)
	rep := &Report{
		Year:      year,
		FetchedAt: fetchedAt,
		Overview: Overview{
			TotalReadTime:          total,
			TotalReadTimeFormatted: formatSeconds(total),
			DayAverageReadTime:     avg,
			DayAverageFormatted:    formatSeconds(avg),
			ReadDays:               days,
			BooksRead:              booksRead,
			BooksFinished:          booksFinished,
			NoteCount:              hl + rv,
			HighlightCount:         hl,
			ReviewCount:            rv,
		},
		Months:          months,
		PeakMonth:       peak,
		Cheer:           cheerText(days, year),
		Favorite:        firstBook(slots, "我的最爱", pickFavorite(data)),
		MostHighlights:  firstBook(slots, "思考最多", mostHighlights(notes)),
		Rarest:          firstBook(slots, "欣赏小众", rarestBook(notes)),
		LateNight:       firstBook(slots, "读到深夜", lateNightBook(notes)),
		Immersed:        firstBook(slots, "最沉浸的", nil),
		FirstRead:       firstBook(slots, "第一本阅读", nil),
		Holidays:        holidayReads(year, notes, data),
		MonthBooks:      monthBooks(notes),
		Categories:      preferCategories(data["preferCategory"]),
		Hours:           hours,
		HoursUnit:       hoursUnit,
		Authors:         preferNamed(data["preferAuthor"]),
		Copyrights:      preferNamed(firstNonNil(data["preferCp"], data["preferCP"], data["preferPublisher"])),
		CopyrightSource: copyrightSource(data),
	}
	return rep, nil
}

func YearsMeta(st *store.Store) (years []int, cached []int, current int, err error) {
	current = store.CalendarYear(time.Now())
	minTs, _, err := st.NoteTimeRange()
	if err != nil {
		return nil, nil, 0, err
	}
	start := 2018
	if minTs > 0 {
		y := time.Unix(minTs, 0).In(shanghai()).Year()
		if y >= 2015 && y < start {
			start = y
		} else if y > start && y <= current {
			start = y
		}
	}
	for y := current; y >= start; y-- {
		years = append(years, y)
	}
	cached, err = st.ListAnnualYears()
	if err != nil {
		return nil, nil, 0, err
	}
	return years, cached, current, nil
}

func parseReadStat(raw any) (read, finished string) {
	for _, item := range conv.AsList(raw) {
		row := conv.AsMap(item)
		if row == nil {
			continue
		}
		label := strings.TrimSpace(asStr(row["stat"]))
		val := strings.TrimSpace(asStr(row["counts"]))
		if val == "" {
			val = strings.TrimSpace(asStr(row["count"]))
		}
		if label == "" {
			continue
		}
		if strings.Contains(label, "读完") {
			if finished == "" {
				finished = val
			}
			continue
		}
		if strings.Contains(label, "读过") || strings.Contains(label, "在读") {
			if read == "" {
				read = val
			}
		}
	}
	return read, finished
}

func parseMonthSeconds(data map[string]any, year int) []MonthBar {
	out := make([]MonthBar, 12)
	for i := 0; i < 12; i++ {
		out[i] = MonthBar{Month: i + 1, Label: fmt.Sprintf("%d月", i+1)}
	}
	raw := data["readTimes"]
	if raw == nil {
		raw = data["dailyReadTimes"]
	}
	add := func(month int, sec int64) {
		if month < 1 || month > 12 {
			return
		}
		out[month-1].Seconds += sec
	}
	switch v := raw.(type) {
	case []any:
		if len(v) == 12 && allNumbers(v) {
			for i, item := range v {
				sec, _ := conv.AsInt64(item)
				out[i].Seconds = sec
			}
			return out
		}
		for i, item := range v {
			if n, ok := conv.AsInt64(item); ok {
				t := time.Date(year, time.January, 1, 0, 0, 0, 0, shanghai()).AddDate(0, 0, i)
				if t.Year() == year {
					add(int(t.Month()), n)
				}
				continue
			}
			row := conv.AsMap(item)
			if row == nil {
				continue
			}
			sec, _ := conv.AsInt64(row["readTime"])
			if sec == 0 {
				sec, _ = conv.AsInt64(row["time"])
			}
			if sec == 0 {
				sec, _ = conv.AsInt64(row["value"])
			}
			month := monthFromRow(row, year)
			add(month, sec)
		}
	default:
		m := conv.AsMap(raw)
		for k, val := range m {
			sec, _ := conv.AsInt64(val)
			ts, ok := conv.AsInt64(k)
			if ok && ts > 0 {
				if ts > 1e12 {
					ts = ts / 1000
				}
				t := time.Unix(ts, 0).In(shanghai())
				if t.Year() == year {
					add(int(t.Month()), sec)
				}
				continue
			}
			if n, err := parseInt(k); err == nil && n >= 1 && n <= 12 {
				add(n, sec)
			}
		}
	}
	return out
}

func monthFromRow(row map[string]any, year int) int {
	if ts, ok := conv.AsInt64(row["baseTime"]); ok && ts > 0 {
		if ts > 1e12 {
			ts /= 1000
		}
		t := time.Unix(ts, 0).In(shanghai())
		if t.Year() == year {
			return int(t.Month())
		}
	}
	s := asStr(row["date"])
	if s == "" {
		s = asStr(row["day"])
	}
	if t, err := time.ParseInLocation("2006-01-02", s, shanghai()); err == nil {
		return int(t.Month())
	}
	if t, err := time.ParseInLocation("2006/1/2", s, shanghai()); err == nil {
		return int(t.Month())
	}
	return 0
}

func parsePreferBooks(raw any) map[string]*BookCard {
	out := map[string]*BookCard{}
	for _, item := range conv.AsList(raw) {
		row := conv.AsMap(item)
		if row == nil {
			continue
		}
		label := strings.TrimSpace(asStr(row["title"]))
		book := conv.AsMap(row["bookInfo"])
		if book == nil {
			book = conv.AsMap(row["book"])
		}
		c := bookCardFromMap(book)
		if c == nil || label == "" {
			continue
		}
		c.Hint = label
		out[label] = c
	}
	return out
}

func firstBook(slots map[string]*BookCard, label string, fallback *BookCard) *BookCard {
	if c := slots[label]; c != nil {
		return c
	}
	return fallback
}

func pickFavorite(data map[string]any) *BookCard {
	var best *BookCard
	var bestTime int64 = -1
	for _, item := range conv.AsList(data["readLongest"]) {
		row := conv.AsMap(item)
		if row == nil {
			continue
		}
		book := conv.AsMap(row["book"])
		if book == nil {
			book = conv.AsMap(row["albumInfo"])
		}
		c := bookCardFromMap(book)
		if c == nil {
			continue
		}
		sec, _ := conv.AsInt64(row["readTime"])
		if sec > bestTime {
			bestTime = sec
			c.Hint = formatSeconds(sec)
			cp := *c
			best = &cp
		}
	}
	return best
}

func mostHighlights(notes []store.YearNote) *BookCard {
	type agg struct {
		card BookCard
		hl   int
		rv   int
	}
	m := map[string]*agg{}
	for _, n := range notes {
		a := m[n.BookID]
		if a == nil {
			a = &agg{card: BookCard{BookID: n.BookID, Title: n.Title, Author: n.Author, Cover: n.Cover}}
			m[n.BookID] = a
		}
		if n.Kind == "highlight" {
			a.hl++
		} else {
			a.rv++
		}
	}
	var best *agg
	for _, a := range m {
		if best == nil || a.hl > best.hl || (a.hl == best.hl && a.rv > best.rv) {
			best = a
		}
	}
	if best == nil || (best.hl == 0 && best.rv == 0) {
		return nil
	}
	best.card.Hint = fmt.Sprintf("划线 %d · 想法 %d", best.hl, best.rv)
	cp := best.card
	return &cp
}

func rarestBook(notes []store.YearNote) *BookCard {
	type agg struct {
		card  BookCard
		count int64
		ok    bool
	}
	m := map[string]*agg{}
	for _, n := range notes {
		a := m[n.BookID]
		if a == nil {
			cnt, ok := readingCount(n.InfoJSON)
			a = &agg{
				card:  BookCard{BookID: n.BookID, Title: n.Title, Author: n.Author, Cover: n.Cover},
				count: cnt,
				ok:    ok,
			}
			m[n.BookID] = a
		}
	}
	var best *agg
	for _, a := range m {
		if !a.ok {
			continue
		}
		if best == nil || a.count < best.count {
			best = a
		}
	}
	if best == nil {
		return nil
	}
	best.card.Hint = fmt.Sprintf("在读 %d 人", best.count)
	cp := best.card
	return &cp
}

func readingCount(infoJSON string) (int64, bool) {
	m := conv.ParseJSONMap(infoJSON)
	if m == nil {
		return 0, false
	}
	for _, k := range []string{"readingCount", "totalReadingCount", "readCount", "readers", "totalCount"} {
		if n, ok := conv.AsInt64(m[k]); ok && n > 0 {
			return n, true
		}
	}
	if nested := conv.AsMap(m["book"]); nested != nil {
		if n, ok := conv.AsInt64(nested["readingCount"]); ok && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func lateNightBook(notes []store.YearNote) *BookCard {
	var best *store.YearNote
	var bestH, bestM, bestS int
	for i := range notes {
		n := notes[i]
		t := time.Unix(n.CreateTime, 0).In(shanghai())
		h := t.Hour()
		if h > 5 {
			continue
		}
		sec := h*3600 + t.Minute()*60 + t.Second()
		bestSec := bestH*3600 + bestM*60 + bestS
		if best == nil || sec > bestSec || (sec == bestSec && n.CreateTime > best.CreateTime) {
			cp := n
			best = &cp
			bestH, bestM, bestS = h, t.Minute(), t.Second()
		}
	}
	if best == nil {
		return nil
	}
	t := time.Unix(best.CreateTime, 0).In(shanghai())
	return &BookCard{
		BookID: best.BookID,
		Title:  best.Title,
		Author: best.Author,
		Cover:  best.Cover,
		Hint:   fmt.Sprintf("%s %02d:%02d", t.Format("1月2日"), t.Hour(), t.Minute()),
	}
}

func holidayReads(year int, notes []store.YearNote, data map[string]any) []HolidayRead {
	daySec := dailySeconds(data, year)
	byDay := map[string]map[string]*aggBook{}
	for _, n := range notes {
		t := time.Unix(n.CreateTime, 0).In(shanghai())
		key := t.Format("2006-01-02")
		if byDay[key] == nil {
			byDay[key] = map[string]*aggBook{}
		}
		a := byDay[key][n.BookID]
		if a == nil {
			a = &aggBook{card: BookCard{BookID: n.BookID, Title: n.Title, Author: n.Author, Cover: n.Cover}}
			byDay[key][n.BookID] = a
		}
		a.n++
	}
	var out []HolidayRead
	for _, h := range holidaysForYear(year) {
		key := h.Date.Format("2006-01-02")
		item := HolidayRead{Name: h.Name, Date: key}
		if books := byDay[key]; len(books) > 0 {
			item.Read = true
			item.BookKnown = true
			var best *aggBook
			for _, a := range books {
				if best == nil || a.n > best.n {
					best = a
				}
			}
			c := best.card
			item.Book = &c
		} else if daySec[key] > 0 {
			item.Read = true
		}
		out = append(out, item)
	}
	return out
}

type aggBook struct {
	card BookCard
	n    int
}

func dailySeconds(data map[string]any, year int) map[string]int64 {
	out := map[string]int64{}
	raw := data["dailyReadTimes"]
	if raw == nil {
		raw = data["readTimes"]
	}
	m := conv.AsMap(raw)
	if m != nil {
		for k, v := range m {
			ts, ok := conv.AsInt64(k)
			if !ok || ts <= 0 {
				continue
			}
			if ts > 1e12 {
				ts /= 1000
			}
			t := time.Unix(ts, 0).In(shanghai())
			if t.Year() != year {
				continue
			}
			sec, _ := conv.AsInt64(v)
			out[t.Format("2006-01-02")] += sec
		}
		if len(out) > 0 {
			return out
		}
	}
	for _, item := range conv.AsList(raw) {
		row := conv.AsMap(item)
		if row == nil {
			continue
		}
		sec, _ := conv.AsInt64(row["readTime"])
		if sec == 0 {
			sec, _ = conv.AsInt64(row["time"])
		}
		date := asStr(row["date"])
		if date == "" {
			if ts, ok := conv.AsInt64(row["baseTime"]); ok && ts > 0 {
				if ts > 1e12 {
					ts /= 1000
				}
				date = time.Unix(ts, 0).In(shanghai()).Format("2006-01-02")
			}
		}
		if date != "" {
			out[date] += sec
		}
	}
	return out
}

func monthBooks(notes []store.YearNote) []MonthBook {
	type key struct {
		month  int
		bookID string
	}
	m := map[key]*aggBook{}
	for _, n := range notes {
		if n.Kind != "highlight" {
			continue
		}
		mo := int(time.Unix(n.CreateTime, 0).In(shanghai()).Month())
		k := key{month: mo, bookID: n.BookID}
		a := m[k]
		if a == nil {
			a = &aggBook{card: BookCard{BookID: n.BookID, Title: n.Title, Author: n.Author, Cover: n.Cover}}
			m[k] = a
		}
		a.n++
	}
	best := map[int]*aggBook{}
	for k, a := range m {
		cur := best[k.month]
		if cur == nil || a.n > cur.n {
			best[k.month] = a
		}
	}
	var out []MonthBook
	for mo := 1; mo <= 12; mo++ {
		a := best[mo]
		if a == nil {
			continue
		}
		out = append(out, MonthBook{Month: mo, Book: a.card, Count: a.n})
	}
	return out
}

func preferCategories(raw any) []NamedCount {
	var out []NamedCount
	for _, item := range conv.AsList(raw) {
		row := conv.AsMap(item)
		if row == nil {
			continue
		}
		name := firstNonEmpty(asStr(row["categoryTitle"]), asStr(row["categoryName"]), asStr(row["name"]), asStr(row["title"]))
		if name == "" {
			continue
		}
		n, _ := conv.AsInt64(row["readingCount"])
		if n == 0 {
			n, _ = conv.AsInt64(row["count"])
		}
		if n <= 0 {
			continue
		}
		out = append(out, NamedCount{Name: name, Count: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func preferHours(raw any, notes []store.YearNote) ([]int64, string) {
	list := conv.AsList(raw)
	if len(list) == 0 {
		hours := make([]int64, 24)
		var n int
		for _, note := range notes {
			h := time.Unix(note.CreateTime, 0).In(shanghai()).Hour()
			if h >= 0 && h < 24 {
				hours[h]++
				n++
			}
		}
		if n == 0 {
			return nil, ""
		}
		return hours, "notes"
	}
	out := make([]int64, 0, 24)
	for _, item := range list {
		n, _ := conv.AsInt64(item)
		if row := conv.AsMap(item); row != nil {
			n, _ = conv.AsInt64(row["value"])
			if n == 0 {
				n, _ = conv.AsInt64(row["readTime"])
			}
		}
		out = append(out, n)
	}
	if len(out) > 24 {
		out = out[:24]
	}
	return out, "seconds"
}

func preferNamed(raw any) []NamedCount {
	var out []NamedCount
	for _, item := range conv.AsList(raw) {
		row := conv.AsMap(item)
		if row == nil {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, NamedCount{Name: s, Count: 1})
			}
			continue
		}
		if info := conv.AsMap(row["copyrightInfo"]); info != nil {
			if n := asStr(info["name"]); n != "" {
				row["name"] = n
			}
		}
		name := firstNonEmpty(asStr(row["name"]), asStr(row["author"]), asStr(row["cpName"]), asStr(row["publisher"]), asStr(row["title"]))
		if name == "" {
			continue
		}
		n, _ := conv.AsInt64(row["count"])
		if n == 0 {
			n, _ = conv.AsInt64(row["readingCount"])
		}
		if n == 0 {
			n = 1
		}
		out = append(out, NamedCount{Name: name, Count: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

func copyrightSource(data map[string]any) string {
	if len(conv.AsList(data["preferCp"])) > 0 || len(conv.AsList(data["preferCP"])) > 0 {
		return "preferCp"
	}
	if len(conv.AsList(data["preferPublisher"])) > 0 {
		return "preferPublisher"
	}
	return ""
}

func bookCardFromMap(book map[string]any) *BookCard {
	if book == nil {
		return nil
	}
	title := asStr(book["title"])
	if title == "" {
		title = asStr(book["name"])
	}
	if title == "" {
		return nil
	}
	id := asStr(book["bookId"])
	if id == "" {
		id, _ = conv.Stringify(book["bookId"])
	}
	return &BookCard{
		BookID: id,
		Title:  title,
		Author: asStr(book["author"]),
		Cover:  asStr(book["cover"]),
	}
}

func cheerText(days int64, year int) string {
	loc := shanghai()
	max := 365
	if time.Date(year, 12, 31, 0, 0, 0, 0, loc).YearDay() == 366 {
		max = 366
	}
	nowY := store.CalendarYear(time.Now())
	if year == nowY {
		max = time.Now().In(loc).YearDay()
	}
	switch {
	case days >= int64(max) && max >= 300:
		return fmt.Sprintf("你 %d 天都在读书，在坚持阅读上，你做到了知行合一，太棒啦", days)
	case days >= 300:
		return fmt.Sprintf("这一年你读了 %d 天，几乎把日子都铺进了书页里，了不起", days)
	case days >= 200:
		return fmt.Sprintf("你有 %d 天在阅读，这份节奏已经很稳，继续把灯亮着就好", days)
	case days >= 100:
		return fmt.Sprintf("你有 %d 天打开过书，习惯正在成形，值得给自己一点掌声", days)
	case days >= 50:
		return fmt.Sprintf("你有 %d 天在读，已经比「想起来再翻」走得更远了", days)
	case days > 0:
		return fmt.Sprintf("这一年你读了 %d 天。次数不在多，翻开的每一页都算数", days)
	default:
		return "这一年的阅读轨迹还很淡，等你再读几页，报告会慢慢丰满起来"
	}
}

func formatSeconds(sec int64) string {
	if sec <= 0 {
		return "0 分钟"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%d 小时 %d 分钟", h, m)
	case h > 0:
		return fmt.Sprintf("%d 小时", h)
	default:
		return fmt.Sprintf("%d 分钟", m)
	}
}

func asStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	s, _ := conv.Stringify(v)
	return s
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func firstNonNil(vs ...any) any {
	for _, v := range vs {
		if v == nil {
			continue
		}
		if list := conv.AsList(v); len(list) > 0 {
			return v
		}
	}
	return nil
}

func allNumbers(list []any) bool {
	for _, item := range list {
		if _, ok := conv.AsInt64(item); !ok {
			return false
		}
	}
	return true
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
