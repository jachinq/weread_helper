package syncjob

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jachin/weread-helper/internal/conv"
	"github.com/jachin/weread-helper/internal/store"
	"github.com/jachin/weread-helper/internal/weread"
)

type Job struct {
	client   *weread.Client
	store    *store.Store
	interval time.Duration

	mu          sync.Mutex
	running     bool
	lastOkAt    int64
	lastError   string
	phase       string
	startedAt   int64
	dirtyTotal  int
	dirtyDone   int
	currentBook string
}

type Status struct {
	State       string `json:"state"`
	LastOkAt    int64  `json:"lastOkAt"`
	LastError   string `json:"lastError"`
	Stale       bool   `json:"stale"`
	Phase       string `json:"phase"`
	StartedAt   int64  `json:"startedAt"`
	ElapsedSec  int64  `json:"elapsedSec"`
	DirtyTotal  int    `json:"dirtyTotal"`
	DirtyDone   int    `json:"dirtyDone"`
	CurrentBook string `json:"currentBookId"`
}

func New(client *weread.Client, st *store.Store, interval time.Duration) *Job {
	j := &Job{client: client, store: st, interval: interval}
	if v, err := st.Meta("last_ok_at"); err == nil && v != "" {
		j.lastOkAt, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, err := st.Meta("last_error"); err == nil {
		j.lastError = v
	}
	return j
}

func (j *Job) ApplyRuntime(interval time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.interval = interval
}

func (j *Job) Status() Status {
	j.mu.Lock()
	defer j.mu.Unlock()
	state := "idle"
	if j.running {
		state = "running"
	}
	stale := !j.running && (j.lastOkAt == 0 || time.Since(time.Unix(j.lastOkAt, 0)) > j.interval)
	var elapsed int64
	if j.running && j.startedAt > 0 {
		elapsed = time.Now().Unix() - j.startedAt
	}
	return Status{
		State:       state,
		LastOkAt:    j.lastOkAt,
		LastError:   j.lastError,
		Stale:       stale,
		Phase:       j.phase,
		StartedAt:   j.startedAt,
		ElapsedSec:  elapsed,
		DirtyTotal:  j.dirtyTotal,
		DirtyDone:   j.dirtyDone,
		CurrentBook: j.currentBook,
	}
}

func (j *Job) setProgress(phase, bookID string, done, total int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if phase != "" {
		j.phase = phase
	}
	j.currentBook = bookID
	if total >= 0 {
		j.dirtyTotal = total
	}
	if done >= 0 {
		j.dirtyDone = done
	}
}

func (j *Job) MaybeStart(force bool) (started bool) {
	st := j.Status()
	if st.State == "running" {
		return false
	}
	if !force && !st.Stale {
		return false
	}
	return j.Start(force)
}

func (j *Job) Start(force bool) bool {
	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		return false
	}
	j.running = true
	j.startedAt = time.Now().Unix()
	j.phase = "starting"
	j.dirtyTotal = 0
	j.dirtyDone = 0
	j.currentBook = ""
	j.mu.Unlock()
	go j.run(force)
	return true
}

func (j *Job) run(force bool) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("sync panic: %v", rec)
			j.mu.Lock()
			j.lastError = fmt.Sprintf("同步异常退出: %v", rec)
			j.mu.Unlock()
			_ = j.store.SetMeta("last_error", fmt.Sprintf("%v", rec))
		}
		j.mu.Lock()
		j.running = false
		j.phase = ""
		j.currentBook = ""
		j.mu.Unlock()
	}()
	log.Printf("sync started force=%v", force)
	j.mu.Lock()
	j.lastError = ""
	j.mu.Unlock()
	if err := j.syncAll(force); err != nil {
		log.Printf("sync failed: %v", err)
		j.mu.Lock()
		j.lastError = err.Error()
		j.mu.Unlock()
		_ = j.store.SetMeta("last_error", err.Error())
		return
	}
	now := time.Now().Unix()
	j.mu.Lock()
	j.lastOkAt = now
	errMsg := j.lastError
	j.mu.Unlock()
	_ = j.store.SetMeta("last_ok_at", strconv.FormatInt(now, 10))
	_ = j.store.SetMeta("last_error", errMsg)
	log.Printf("sync finished")
}

func (j *Job) syncAll(force bool) error {
	j.setProgress("notebooks", "", 0, 0)
	if err := j.syncNotebooks(force); err != nil {
		return fmt.Errorf("notebooks: %w", err)
	}
	j.setProgress("shelf", "", -1, -1)
	if err := j.syncShelf(); err != nil {
		return fmt.Errorf("shelf: %w", err)
	}
	j.setProgress("stats", "", -1, -1)
	if err := j.syncStats(); err != nil {
		return fmt.Errorf("stats: %w", err)
	}
	j.setProgress("done", "", -1, -1)
	return nil
}

func (j *Job) syncNotebooks(force bool) error {
	var lastSort int64
	seen := make([]string, 0)
	dirty := make([]string, 0)
	for pageNo := 0; pageNo < 200; pageNo++ {
		page, err := j.client.Notebooks(100, lastSort)
		if err != nil {
			return err
		}
		list := conv.AsList(page["books"])
		if len(list) == 0 {
			break
		}
		var pageLast int64
		for _, item := range list {
			m := conv.AsMap(item)
			if m == nil {
				continue
			}
			nested := conv.AsMap(m["book"])
			bookID, _ := conv.Stringify(m["bookId"])
			if bookID == "" && nested != nil {
				bookID, _ = conv.Stringify(nested["bookId"])
			}
			if bookID == "" {
				continue
			}
			existing, err := j.store.GetBook(bookID)
			if err != nil {
				return err
			}
			nb := notebookBook(bookID, m, nested)
			isDirty := force || existing == nil || existing.NotesSyncedAt == 0 ||
				existing.NoteCount != nb.NoteCount ||
				existing.ReviewCount != nb.ReviewCount ||
				existing.BookmarkCount != nb.BookmarkCount ||
				existing.Sort != nb.Sort ||
				existing.ReadingProgress != nb.ReadingProgress
			if err := j.store.UpsertNotebook(nb); err != nil {
				return err
			}
			seen = append(seen, bookID)
			if isDirty {
				dirty = append(dirty, bookID)
			}
			pageLast, _ = conv.AsInt64(m["sort"])
		}
		hasMore, _ := conv.AsInt64(page["hasMore"])
		if hasMore == 0 || pageLast == 0 || pageLast == lastSort {
			break
		}
		lastSort = pageLast
	}
	if err := j.store.ClearNotebooksExcept(seen); err != nil {
		return err
	}
	j.setProgress("notes", "", 0, len(dirty))
	var errs []string
	for i, id := range dirty {
		j.setProgress("notes", id, i+1, len(dirty))
		log.Printf("sync book %d/%d %s", i+1, len(dirty), id)
		if err := j.refreshBookNotes(id); err != nil {
			errs = append(errs, id+": "+err.Error())
			log.Printf("sync book %s: %v", id, err)
		}
	}
	j.setProgress("notes", "", len(dirty), len(dirty))
	if len(errs) > 0 {
		msg := "部分书籍笔记失败: " + strings.Join(errs, "; ")
		_ = j.store.SetMeta("last_error", msg)
		j.mu.Lock()
		j.lastError = msg
		j.mu.Unlock()
	}
	return nil
}

func notebookBook(bookID string, row, nested map[string]any) *store.Book {
	b := &store.Book{BookID: bookID, InNotebooks: true}
	src := nested
	if src == nil {
		src = map[string]any{}
	}
	b.Title, _ = src["title"].(string)
	if b.Title == "" {
		b.Title, _ = row["title"].(string)
	}
	b.Author, _ = src["author"].(string)
	if b.Author == "" {
		b.Author, _ = row["author"].(string)
	}
	b.Cover, _ = src["cover"].(string)
	if b.Cover == "" {
		b.Cover, _ = row["cover"].(string)
	}
	b.ReviewCount, _ = conv.AsInt64(row["reviewCount"])
	b.NoteCount, _ = conv.AsInt64(row["noteCount"])
	b.BookmarkCount, _ = conv.AsInt64(row["bookmarkCount"])
	b.ReadingProgress, _ = conv.AsInt64(row["readingProgress"])
	if b.ReadingProgress == 0 {
		b.ReadingProgress, _ = conv.AsInt64(src["readingProgress"])
	}
	b.Sort, _ = conv.AsInt64(row["sort"])
	b.MarkedStatus, _ = conv.AsInt64(row["markedStatus"])
	return b
}

func (j *Job) refreshBookNotes(bookID string) error {
	chaptersRaw, err := j.client.Chapters(bookID)
	if err != nil {
		return err
	}
	highlightsRaw, err := j.client.Highlights(bookID)
	if err != nil {
		return err
	}
	reviewsRaw := make([]any, 0)
	var synckey int64
	for pageNo := 0; pageNo < 50; pageNo++ {
		page, err := j.client.MyReviews(bookID, 100, synckey)
		if err != nil {
			return err
		}
		if list := conv.AsList(page["reviews"]); list != nil {
			reviewsRaw = append(reviewsRaw, list...)
		}
		hasMore, _ := conv.AsInt64(page["hasMore"])
		next, _ := conv.AsInt64(page["synckey"])
		if hasMore == 0 || next == 0 || next == synckey {
			break
		}
		synckey = next
	}

	chapters := parseChapters(bookID, chaptersRaw, highlightsRaw)
	highlights := parseHighlights(bookID, highlightsRaw)
	reviews := parseReviews(bookID, reviewsRaw)
	if err := j.store.ReplaceNotes(bookID, chapters, highlights, reviews); err != nil {
		return err
	}

	fields := map[string]any{}
	if info, err := j.client.BookInfo(bookID); err == nil && info != nil {
		fields["info_json"] = conv.JSONText(info)
		if t, _ := info["title"].(string); t != "" {
			fields["title"] = t
		}
		if a, _ := info["author"].(string); a != "" {
			fields["author"] = a
		}
		if c, _ := info["cover"].(string); c != "" {
			fields["cover"] = c
		}
		if intro, _ := info["intro"].(string); intro != "" {
			fields["intro"] = intro
		}
		if cat, _ := info["category"].(string); cat != "" {
			fields["category"] = cat
		}
		if p, _ := info["publisher"].(string); p != "" {
			fields["publisher"] = p
		}
		if isbn, ok := conv.Stringify(info["isbn"]); ok {
			fields["isbn"] = isbn
		}
	}
	if book := conv.AsMap(highlightsRaw["book"]); book != nil && fields["info_json"] == nil {
		fields["info_json"] = conv.JSONText(book)
	}
	if progress, err := j.client.Progress(bookID); err == nil && progress != nil {
		fields["progress_json"] = conv.JSONText(progress)
		if p, ok := conv.AsInt64(progress["progress"]); ok {
			fields["reading_progress"] = p
		}
	}
	if len(fields) > 0 {
		return j.store.MergeBookFields(bookID, fields)
	}
	return nil
}

func parseChapters(bookID string, chaptersRaw, highlightsRaw map[string]any) []store.Chapter {
	seen := map[int64]store.Chapter{}
	order := make([]int64, 0)
	add := func(m map[string]any) {
		uid, ok := conv.AsInt64(m["chapterUid"])
		if !ok || uid == 0 {
			return
		}
		title, _ := m["title"].(string)
		idx, _ := conv.AsInt64(m["chapterIdx"])
		if prev, exists := seen[uid]; exists {
			if prev.Title == "" {
				prev.Title = title
			}
			if prev.ChapterIdx == 0 {
				prev.ChapterIdx = idx
			}
			seen[uid] = prev
			return
		}
		order = append(order, uid)
		seen[uid] = store.Chapter{BookID: bookID, ChapterUID: uid, Title: title, ChapterIdx: idx}
	}
	walkChapterLists(chaptersRaw, add)
	walkChapterLists(highlightsRaw, add)
	out := make([]store.Chapter, 0, len(order))
	for _, uid := range order {
		out = append(out, seen[uid])
	}
	return out
}

func walkChapterLists(raw map[string]any, add func(map[string]any)) {
	if raw == nil {
		return
	}
	if list := conv.AsList(raw["chapters"]); list != nil {
		for _, item := range list {
			if m := conv.AsMap(item); m != nil {
				add(m)
			}
		}
	}
	if list := conv.AsList(raw["data"]); list != nil {
		for _, item := range list {
			m := conv.AsMap(item)
			if m == nil {
				continue
			}
			inner := conv.AsList(m["chapters"])
			for _, ch := range inner {
				if cm := conv.AsMap(ch); cm != nil {
					add(cm)
				}
			}
		}
	}
}

func parseHighlights(bookID string, raw map[string]any) []store.Highlight {
	var out []store.Highlight
	for _, item := range conv.AsList(raw["updated"]) {
		m := conv.AsMap(item)
		if m == nil {
			continue
		}
		id, _ := conv.Stringify(m["bookmarkId"])
		text, _ := m["markText"].(string)
		uid, _ := conv.AsInt64(m["chapterUid"])
		ct, _ := conv.AsInt64(m["createTime"])
		rg, _ := m["range"].(string)
		out = append(out, store.Highlight{
			BookmarkID: id,
			BookID:     bookID,
			ChapterUID: uid,
			MarkText:   text,
			CreateTime: ct,
			Range:      rg,
			ColorStyle: conv.JSONText(m["colorStyle"]),
		})
	}
	return out
}

func parseReviews(bookID string, list []any) []store.Review {
	var out []store.Review
	for _, item := range list {
		outer := conv.AsMap(item)
		if outer == nil {
			continue
		}
		inner := outer
		if nested := conv.AsMap(outer["review"]); nested != nil {
			inner = nested
		}
		id, _ := conv.Stringify(inner["reviewId"])
		content, _ := inner["content"].(string)
		abstract, _ := inner["abstract"].(string)
		name, _ := inner["chapterName"].(string)
		uid, _ := conv.AsInt64(inner["chapterUid"])
		ct, _ := conv.AsInt64(inner["createTime"])
		rg, _ := inner["range"].(string)
		out = append(out, store.Review{
			ReviewID:    id,
			BookID:      bookID,
			ChapterUID:  uid,
			ChapterName: name,
			Content:     content,
			Abstract:    abstract,
			CreateTime:  ct,
			Star:        conv.JSONText(inner["star"]),
			Range:       rg,
		})
	}
	return out
}

func (j *Job) syncShelf() error {
	data, err := j.client.Shelf()
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	for _, item := range conv.AsList(data["books"]) {
		m := conv.AsMap(item)
		if m == nil {
			continue
		}
		id, _ := conv.Stringify(m["bookId"])
		if id == "" {
			continue
		}
		top, _ := conv.AsInt64(m["isTop"])
		finish, _ := conv.AsInt64(m["finishReading"])
		secret, _ := conv.AsInt64(m["secret"])
		readAt, _ := conv.AsInt64(m["readUpdateTime"])
		title, _ := m["title"].(string)
		author, _ := m["author"].(string)
		cover, _ := m["cover"].(string)
		cat, _ := m["category"].(string)
		b := &store.Book{
			BookID:         id,
			Title:          title,
			Author:         author,
			Cover:          cover,
			Category:       cat,
			IsOnShelf:      true,
			IsTop:          top != 0,
			FinishReading:  finish != 0,
			Secret:         secret != 0,
			ReadUpdateTime: readAt,
		}
		if err := j.store.UpsertShelf(b); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	return j.store.SetOnShelf(ids)
}

func (j *Job) syncStats() error {
	for _, mode := range []string{"weekly", "monthly", "annually", "overall"} {
		data, err := j.client.ReadStats(mode, 0)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if err := j.store.PutStats(mode, string(raw)); err != nil {
			return err
		}
	}
	return nil
}
