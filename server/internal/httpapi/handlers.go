package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/conv"
	"github.com/jachin/weread-helper/internal/report"
	"github.com/jachin/weread-helper/internal/store"
	"github.com/jachin/weread-helper/internal/syncjob"
	"github.com/jachin/weread-helper/internal/weread"
)

type Server struct {
	store     *store.Store
	job       *syncjob.Job
	client    *weread.Client
	encKey    []byte
	pickMu    sync.Mutex
	pickDate  string
	pickItems []store.RandomHighlight
}

func New(st *store.Store, job *syncjob.Job, client *weread.Client, encKey []byte) *Server {
	return &Server{store: st, job: job, client: client, encKey: encKey}
}

func (s *Server) Register(r *gin.Engine) {
	api := r.Group("/api")
	api.GET("/health", s.health)
	api.GET("/notebooks", s.notebooks)
	api.GET("/highlights/random", s.randomHighlights)
	api.POST("/highlights/random", s.refreshHighlights)
	api.GET("/books/:bookId", s.book)
	api.GET("/books/:bookId/notes", s.notes)
	api.GET("/stats", s.stats)
	api.POST("/stats/fetch", s.statsFetch)
	api.GET("/report/years", s.reportYears)
	api.GET("/report", s.reportGet)
	api.POST("/report/fetch", s.reportFetch)
	api.GET("/shelf", s.shelf)
	api.GET("/sync/status", s.syncStatus)
	api.POST("/sync", s.syncStart)
	api.GET("/settings", s.settingsGet)
	api.PUT("/settings", s.settingsPut)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) randomHighlights(c *gin.Context) {
	s.writePicks(c, false)
}

func (s *Server) refreshHighlights(c *gin.Context) {
	s.writePicks(c, true)
}

func (s *Server) writePicks(c *gin.Context, refresh bool) {
	items, date, err := s.todayPicks(refresh)
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, h := range items {
		out = append(out, gin.H{
			"bookmarkId": h.BookmarkID,
			"bookId":     h.BookID,
			"markText":   h.MarkText,
			"createTime": h.CreateTime,
			"title":      h.Title,
			"author":     h.Author,
			"cover":      h.Cover,
		})
	}
	c.JSON(http.StatusOK, gin.H{"date": date, "items": out})
}

func (s *Server) todayPicks(refresh bool) ([]store.RandomHighlight, string, error) {
	today := time.Now().Format("2006-01-02")
	s.pickMu.Lock()
	defer s.pickMu.Unlock()
	if !refresh && s.pickDate == today && s.pickItems != nil {
		return s.pickItems, today, nil
	}
	items, err := s.store.RandomHighlights(5)
	if err != nil {
		return nil, today, err
	}
	if refresh || len(items) > 0 {
		s.pickDate = today
		s.pickItems = items
	}
	return items, today, nil
}

func (s *Server) notebooks(c *gin.Context) {
	count := queryInt(c, "count", 40)
	lastSort := queryInt64(c, "lastSort", 0)
	q := strings.TrimSpace(c.Query("q"))
	books, totalBooks, totalNotes, hasMore, err := s.store.ListNotebooks(count, lastSort, q)
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(books))
	for _, b := range books {
		out = append(out, gin.H{
			"bookId":          b.BookID,
			"reviewCount":     b.ReviewCount,
			"noteCount":       b.NoteCount,
			"bookmarkCount":   b.BookmarkCount,
			"readingProgress": b.ReadingProgress,
			"markedStatus":    b.MarkedStatus,
			"sort":            b.Sort,
			"book": gin.H{
				"bookId": b.BookID,
				"title":  b.Title,
				"author": b.Author,
				"cover":  b.Cover,
			},
		})
	}
	hasMoreN := 0
	if hasMore {
		hasMoreN = 1
	}
	c.JSON(http.StatusOK, gin.H{
		"books":          out,
		"totalBookCount": totalBooks,
		"totalNoteCount": totalNotes,
		"hasMore":        hasMoreN,
	})
}

func (s *Server) book(c *gin.Context) {
	bookID := c.Param("bookId")
	b, err := s.store.GetBook(bookID)
	if err != nil {
		writeErr(c, err)
		return
	}
	if b == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "本地无此书，请先同步"})
		return
	}
	info := conv.ParseJSONMap(b.InfoJSON)
	if info == nil {
		info = map[string]any{
			"bookId": b.BookID,
			"title":  b.Title,
			"author": b.Author,
			"cover":  b.Cover,
			"intro":  b.Intro,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"book":     info,
		"progress": conv.ParseJSONMap(b.ProgressJSON),
	})
}

func (s *Server) notes(c *gin.Context) {
	bookID := c.Param("bookId")
	b, err := s.store.GetBook(bookID)
	if err != nil {
		writeErr(c, err)
		return
	}
	if b == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "本地无此书，请先同步"})
		return
	}
	chs, err := s.store.ListChapters(bookID)
	if err != nil {
		writeErr(c, err)
		return
	}
	hls, err := s.store.ListHighlights(bookID)
	if err != nil {
		writeErr(c, err)
		return
	}
	revs, err := s.store.ListReviews(bookID)
	if err != nil {
		writeErr(c, err)
		return
	}

	chapterList := make([]any, 0, len(chs))
	for _, ch := range chs {
		chapterList = append(chapterList, map[string]any{
			"chapterUid": ch.ChapterUID,
			"title":      ch.Title,
			"chapterIdx": ch.ChapterIdx,
		})
	}
	updated := make([]any, 0, len(hls))
	for _, h := range hls {
		updated = append(updated, map[string]any{
			"bookmarkId": h.BookmarkID,
			"chapterUid": h.ChapterUID,
			"markText":   h.MarkText,
			"createTime": h.CreateTime,
			"range":      h.Range,
			"colorStyle": conv.ParseJSONAny(h.ColorStyle),
		})
	}
	reviews := make([]any, 0, len(revs))
	for _, r := range revs {
		reviews = append(reviews, map[string]any{
			"review": map[string]any{
				"reviewId":    r.ReviewID,
				"chapterUid":  r.ChapterUID,
				"chapterName": r.ChapterName,
				"content":     r.Content,
				"abstract":    r.Abstract,
				"createTime":  r.CreateTime,
				"range":       r.Range,
				"star":        conv.ParseJSONAny(r.Star),
			},
		})
	}

	chaptersRaw := map[string]any{"chapters": chapterList}
	highlightsRaw := map[string]any{"updated": updated, "chapters": chapterList}
	grouped := groupNotes(chaptersRaw, highlightsRaw, reviews)
	book := conv.ParseJSONMap(b.InfoJSON)
	if book == nil {
		book = map[string]any{"bookId": b.BookID, "title": b.Title, "author": b.Author, "cover": b.Cover}
	}
	if intro := strings.TrimSpace(b.Intro); intro != "" {
		if existing, _ := book["intro"].(string); strings.TrimSpace(existing) == "" {
			book["intro"] = intro
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"bookId":   bookID,
		"book":     book,
		"chapters": grouped,
	})
}

func (s *Server) stats(c *gin.Context) {
	mode := c.DefaultQuery("mode", "monthly")
	switch mode {
	case "weekly", "monthly", "annually", "overall":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode 必须是 weekly/monthly/annually/overall"})
		return
	}
	period, ok, errMsg := parseStatsPeriod(mode, c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	var payload string
	var err error
	if period != nil {
		payload, _, err = s.store.GetStatsYear(mode, *period)
	} else {
		payload, _, err = s.store.GetStats(mode)
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	if payload == "" {
		if period != nil {
			c.JSON(http.StatusNotFound, statsMissing(mode, *period))
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "本地尚无统计数据，请先同步"})
		return
	}
	writeStatsJSON(c, mode, payload, period)
}

func (s *Server) statsFetch(c *gin.Context) {
	mode := c.DefaultQuery("mode", "")
	switch mode {
	case "weekly", "monthly", "annually":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode 必须是 weekly/monthly/annually"})
		return
	}
	period, ok, errMsg := parseStatsPeriod(mode, c)
	if !ok || period == nil {
		if errMsg == "" {
			errMsg = "缺少 year / month / week"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	var err error
	switch mode {
	case "annually":
		err = report.PullAnnual(s.client, s.store, int(*period))
	case "monthly":
		err = report.PullMonthly(s.client, s.store, *period)
	case "weekly":
		err = report.PullWeekly(s.client, s.store, *period)
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	payload, _, err := s.store.GetStatsYear(mode, *period)
	if err != nil {
		writeErr(c, err)
		return
	}
	if payload == "" {
		c.JSON(http.StatusNotFound, statsMissing(mode, *period))
		return
	}
	writeStatsJSON(c, mode, payload, period)
}

func parseStatsPeriod(mode string, c *gin.Context) (*int64, bool, string) {
	switch mode {
	case "annually":
		raw := strings.TrimSpace(c.Query("year"))
		if raw == "" {
			return nil, true, ""
		}
		y, ok := parseReportYear(raw)
		if !ok {
			return nil, false, "year 无效"
		}
		v := int64(y)
		return &v, true, ""
	case "monthly":
		raw := strings.TrimSpace(c.Query("month"))
		if raw == "" {
			return nil, true, ""
		}
		key, ok := parseStatsMonth(raw)
		if !ok {
			return nil, false, "month 无效"
		}
		return &key, true, ""
	case "weekly":
		raw := strings.TrimSpace(c.Query("week"))
		if raw == "" {
			return nil, true, ""
		}
		key, ok := parseStatsWeek(raw)
		if !ok {
			return nil, false, "week 无效"
		}
		return &key, true, ""
	default:
		return nil, true, ""
	}
}

func parseStatsMonth(raw string) (int64, bool) {
	t, err := time.ParseInLocation("2006-01", raw, store.Shanghai())
	if err != nil {
		return 0, false
	}
	cur := store.MonthKey(time.Now())
	key := store.MonthKey(t)
	if key < 201501 || key > cur {
		return 0, false
	}
	return key, true
}

func parseStatsWeek(raw string) (int64, bool) {
	loc := store.Shanghai()
	var mon time.Time
	if strings.Contains(raw, "W") || strings.Contains(raw, "w") {
		var y, w int
		n, err := fmt.Sscanf(strings.ToUpper(raw), "%d-W%d", &y, &w)
		if err != nil || n != 2 || w < 1 || w > 53 || y < 2015 {
			return 0, false
		}
		mon = isoWeekMonday(y, w)
	} else {
		t, err := time.ParseInLocation("2006-01-02", raw, loc)
		if err != nil {
			return 0, false
		}
		mon = store.WeekMonday(t)
	}
	if mon.IsZero() {
		return 0, false
	}
	key := store.WeekKey(mon)
	nowKey := store.WeekKey(time.Now())
	if key < 20150101 || key > nowKey {
		return 0, false
	}
	return key, true
}

func isoWeekMonday(year, week int) time.Time {
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, store.Shanghai())
	mon := store.WeekMonday(jan4)
	return mon.AddDate(0, 0, (week-1)*7)
}

func statsMissing(mode string, period int64) gin.H {
	h := gin.H{"missing": true, "mode": mode, "error": "本地尚无该周期统计快照"}
	switch mode {
	case "annually":
		h["year"] = int(period)
	case "monthly":
		t := store.TimeFromMonthKey(period)
		h["month"] = t.Format("2006-01")
	case "weekly":
		t := store.TimeFromWeekKey(period)
		iy, iw := t.ISOWeek()
		h["week"] = fmt.Sprintf("%d-W%02d", iy, iw)
		h["weekStart"] = t.Format("2006-01-02")
	}
	return h
}

func writeStatsJSON(c *gin.Context, mode, payload string, period *int64) {
	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		writeErr(c, err)
		return
	}
	total, _ := conv.AsInt64(data["totalReadTime"])
	avg, avgOK := conv.AsInt64(data["dayAverageReadTime"])
	days, _ := conv.AsInt64(data["readDays"])
	// 官方 /readdata/detail 在 overall 下不返回 dayAverageReadTime
	if (!avgOK || avg <= 0) && total > 0 && days > 0 {
		avg = total / days
		data["dayAverageReadTime"] = avg
	}
	data["totalReadTimeFormatted"] = formatSeconds(total)
	data["dayAverageReadTimeFormatted"] = formatSeconds(avg)
	data["mode"] = mode
	key := int64(0)
	if period != nil {
		key = *period
	} else {
		key = store.PeriodKeyFromPayload(mode, payload, time.Now())
	}
	switch mode {
	case "annually":
		data["year"] = int(key)
	case "monthly":
		t := store.TimeFromMonthKey(key)
		if !t.IsZero() {
			data["month"] = t.Format("2006-01")
			data["year"] = t.Year()
		}
	case "weekly":
		t := store.TimeFromWeekKey(key)
		if !t.IsZero() {
			iy, iw := t.ISOWeek()
			data["week"] = fmt.Sprintf("%d-W%02d", iy, iw)
			data["weekStart"] = t.Format("2006-01-02")
			data["year"] = t.Year()
		}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, data)
}

func (s *Server) reportYears(c *gin.Context) {
	years, cached, current, err := report.YearsMeta(s.store)
	if err != nil {
		writeErr(c, err)
		return
	}
	if years == nil {
		years = []int{}
	}
	if cached == nil {
		cached = []int{}
	}
	c.JSON(http.StatusOK, gin.H{"years": years, "cached": cached, "current": current})
}

func (s *Server) reportGet(c *gin.Context) {
	year, ok := parseReportYear(c.Query("year"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year 无效"})
		return
	}
	rep, err := report.Build(s.store, year)
	if err != nil {
		var miss *report.MissingSnapshotError
		if errors.As(err, &miss) {
			c.JSON(http.StatusNotFound, gin.H{"missing": true, "year": miss.Year, "error": miss.Error()})
			return
		}
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (s *Server) reportFetch(c *gin.Context) {
	year, ok := parseReportYear(c.Query("year"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year 无效"})
		return
	}
	if err := report.PullAnnual(s.client, s.store, year); err != nil {
		writeErr(c, err)
		return
	}
	rep, err := report.Build(s.store, year)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, rep)
}

func parseReportYear(raw string) (int, bool) {
	cur := store.CalendarYear(time.Now())
	if strings.TrimSpace(raw) == "" {
		return cur, true
	}
	y, err := strconv.Atoi(raw)
	if err != nil || y < 2015 || y > cur+1 {
		return 0, false
	}
	return y, true
}

func (s *Server) shelf(c *gin.Context) {
	books, err := s.store.ListShelf()
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(books))
	for _, b := range books {
		out = append(out, gin.H{
			"bookId":          b.BookID,
			"title":           b.Title,
			"author":          b.Author,
			"cover":           b.Cover,
			"category":        b.Category,
			"readingProgress": b.ReadingProgress,
			"isTop":           b.IsTop,
			"finishReading":   b.FinishReading,
			"readUpdateTime":  b.ReadUpdateTime,
		})
	}
	c.JSON(http.StatusOK, gin.H{"books": out, "bookCount": len(out)})
}

func (s *Server) syncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.job.Status())
}

func (s *Server) syncStart(c *gin.Context) {
	force := c.Query("force") == "1" || c.Query("force") == "true"
	started := s.job.Start(force)
	st := s.job.Status()
	c.JSON(http.StatusOK, gin.H{
		"started":       started,
		"state":         st.State,
		"lastOkAt":      st.LastOkAt,
		"lastError":     st.LastError,
		"stale":         st.Stale,
		"phase":         st.Phase,
		"startedAt":     st.StartedAt,
		"elapsedSec":    st.ElapsedSec,
		"dirtyTotal":    st.DirtyTotal,
		"dirtyDone":     st.DirtyDone,
		"currentBookId": st.CurrentBook,
	})
}

type highlightItem struct {
	BookmarkID string `json:"bookmarkId"`
	MarkText   string `json:"markText"`
	CreateTime int64  `json:"createTime"`
	ColorStyle any    `json:"colorStyle,omitempty"`
	Range      string `json:"range,omitempty"`
}

type reviewItem struct {
	ReviewID   string `json:"reviewId"`
	Content    string `json:"content"`
	Abstract   string `json:"abstract"`
	CreateTime int64  `json:"createTime"`
	Star       any    `json:"star,omitempty"`
}

type chapterGroup struct {
	ChapterUID int64           `json:"chapterUid"`
	Title      string          `json:"title"`
	ChapterIdx int64           `json:"chapterIdx"`
	Highlights []highlightItem `json:"highlights"`
	Reviews    []reviewItem    `json:"reviews"`
}

func groupNotes(chaptersRaw, highlightsRaw map[string]any, reviews []any) []chapterGroup {
	titleByUID := map[int64]chapterGroup{}
	order := make([]int64, 0)

	addChapter := func(uid, idx int64, title string) {
		if uid == 0 {
			return
		}
		if g, exists := titleByUID[uid]; exists {
			if g.Title == "" {
				g.Title = title
			}
			if g.ChapterIdx == 0 {
				g.ChapterIdx = idx
			}
			titleByUID[uid] = g
			return
		}
		order = append(order, uid)
		titleByUID[uid] = chapterGroup{
			ChapterUID: uid,
			Title:      title,
			ChapterIdx: idx,
			Highlights: []highlightItem{},
			Reviews:    []reviewItem{},
		}
	}

	if list, ok := chaptersRaw["chapters"].([]any); ok {
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			uid, _ := conv.AsInt64(m["chapterUid"])
			idx, _ := conv.AsInt64(m["chapterIdx"])
			title, _ := m["title"].(string)
			addChapter(uid, idx, title)
		}
	}

	if list, ok := highlightsRaw["chapters"].([]any); ok {
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			uid, _ := conv.AsInt64(m["chapterUid"])
			idx, _ := conv.AsInt64(m["chapterIdx"])
			title, _ := m["title"].(string)
			addChapter(uid, idx, title)
		}
	}

	ensure := func(uid int64) chapterGroup {
		g, ok := titleByUID[uid]
		if !ok {
			order = append(order, uid)
			g = chapterGroup{
				ChapterUID: uid,
				Title:      "未分类章节",
				Highlights: []highlightItem{},
				Reviews:    []reviewItem{},
			}
			titleByUID[uid] = g
		}
		return g
	}

	if list, ok := highlightsRaw["updated"].([]any); ok {
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			uid, _ := conv.AsInt64(m["chapterUid"])
			g := ensure(uid)
			id, _ := conv.Stringify(m["bookmarkId"])
			text, _ := m["markText"].(string)
			ct, _ := conv.AsInt64(m["createTime"])
			rangeStr, _ := m["range"].(string)
			g.Highlights = append(g.Highlights, highlightItem{
				BookmarkID: id,
				MarkText:   text,
				CreateTime: ct,
				ColorStyle: m["colorStyle"],
				Range:      rangeStr,
			})
			titleByUID[uid] = g
		}
	}

	for _, item := range reviews {
		outer, ok := item.(map[string]any)
		if !ok {
			continue
		}
		inner := outer
		if nested, ok := outer["review"].(map[string]any); ok {
			inner = nested
		}
		uid, _ := conv.AsInt64(inner["chapterUid"])
		g := ensure(uid)
		if g.Title == "未分类章节" {
			if name, _ := inner["chapterName"].(string); name != "" {
				g.Title = name
			}
		}
		id, _ := conv.Stringify(inner["reviewId"])
		content, _ := inner["content"].(string)
		abstract, _ := inner["abstract"].(string)
		ct, _ := conv.AsInt64(inner["createTime"])
		g.Reviews = append(g.Reviews, reviewItem{
			ReviewID:   id,
			Content:    content,
			Abstract:   abstract,
			CreateTime: ct,
			Star:       inner["star"],
		})
		titleByUID[uid] = g
	}

	out := make([]chapterGroup, 0, len(order))
	for _, uid := range order {
		g := titleByUID[uid]
		if len(g.Highlights) == 0 && len(g.Reviews) == 0 {
			continue
		}
		out = append(out, g)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ChapterIdx == out[j].ChapterIdx {
			return out[i].ChapterUID < out[j].ChapterUID
		}
		return out[i].ChapterIdx < out[j].ChapterIdx
	})
	return out
}

func writeErr(c *gin.Context, err error) {
	var apiErr *weread.APIError
	if errors.As(err, &apiErr) {
		status := http.StatusBadGateway
		if apiErr.Code == 426 {
			status = http.StatusUpgradeRequired
		}
		c.JSON(status, gin.H{
			"error":   apiErr.Message,
			"code":    apiErr.Code,
			"upgrade": apiErr.Upgrade,
		})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
}

func queryInt(c *gin.Context, key string, fallback int) int {
	v := c.Query(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func queryInt64(c *gin.Context, key string, fallback int64) int64 {
	v := c.Query(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func formatSeconds(sec int64) string {
	if sec <= 0 {
		return "0 分钟"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	switch {
	case h > 0 && m > 0:
		return strconv.FormatInt(h, 10) + " 小时 " + strconv.FormatInt(m, 10) + " 分钟"
	case h > 0:
		return strconv.FormatInt(h, 10) + " 小时"
	default:
		return strconv.FormatInt(m, 10) + " 分钟"
	}
}
