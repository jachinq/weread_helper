package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/conv"
	"github.com/jachin/weread-helper/internal/store"
	"github.com/jachin/weread-helper/internal/syncjob"
	"github.com/jachin/weread-helper/internal/weread"
)

type Server struct {
	store     *store.Store
	job       *syncjob.Job
	pickMu    sync.Mutex
	pickDate  string
	pickItems []store.RandomHighlight
}

func New(st *store.Store, job *syncjob.Job) *Server {
	return &Server{store: st, job: job}
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
	api.GET("/shelf", s.shelf)
	api.GET("/sync/status", s.syncStatus)
	api.POST("/sync", s.syncStart)
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
	s.job.MaybeStart(false)
	count := queryInt(c, "count", 40)
	lastSort := queryInt64(c, "lastSort", 0)
	books, totalBooks, totalNotes, hasMore, err := s.store.ListNotebooks(count, lastSort)
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
	s.job.MaybeStart(false)
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
	s.job.MaybeStart(false)
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
	c.JSON(http.StatusOK, gin.H{
		"bookId":   bookID,
		"book":     book,
		"chapters": grouped,
	})
}

func (s *Server) stats(c *gin.Context) {
	s.job.MaybeStart(false)
	mode := c.DefaultQuery("mode", "monthly")
	switch mode {
	case "weekly", "monthly", "annually", "overall":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode 必须是 weekly/monthly/annually/overall"})
		return
	}
	payload, _, err := s.store.GetStats(mode)
	if err != nil {
		writeErr(c, err)
		return
	}
	if payload == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "本地尚无统计数据，请先同步"})
		return
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		writeErr(c, err)
		return
	}
	total, _ := conv.AsInt64(data["totalReadTime"])
	avg, _ := conv.AsInt64(data["dayAverageReadTime"])
	data["totalReadTimeFormatted"] = formatSeconds(total)
	data["dayAverageReadTimeFormatted"] = formatSeconds(avg)
	data["mode"] = mode
	c.JSON(http.StatusOK, data)
}

func (s *Server) shelf(c *gin.Context) {
	s.job.MaybeStart(false)
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
