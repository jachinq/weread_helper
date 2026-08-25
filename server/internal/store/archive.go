package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"
)

type NoteSearchHit struct {
	Kind         string
	BookmarkID   string
	ReviewID     string
	BookID       string
	Title        string
	Author       string
	Cover        string
	ChapterUID   int64
	ChapterTitle string
	MarkText     string
	Content      string
	Abstract     string
	CreateTime   int64
	Starred      bool
}

func clampSearchLimit(limit int) int {
	if limit <= 0 {
		return 40
	}
	if limit > 80 {
		return 80
	}
	return limit
}

func (s *Store) SearchNotes(query, kind string, limit int) ([]NoteSearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []NoteSearchHit{}, nil
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = "all"
	}
	limit = clampSearchLimit(limit)

	const highlightSQL = `
SELECT 'highlight' AS kind, h.bookmark_id AS bookmark_id, '' AS review_id, h.book_id, h.chapter_uid,
       COALESCE(c.title, '') AS chapter_title, h.mark_text AS mark_text, '' AS content, '' AS abstract,
       h.create_time AS create_time, b.title, b.author, b.cover,
       CASE WHEN s.bookmark_id IS NULL THEN 0 ELSE 1 END AS starred
FROM highlights h
JOIN books b ON b.book_id = h.book_id
LEFT JOIN chapters c ON c.book_id = h.book_id AND c.chapter_uid = h.chapter_uid
LEFT JOIN starred_highlights s ON s.bookmark_id = h.bookmark_id
WHERE TRIM(h.mark_text) != ''
  AND (instr(lower(h.mark_text), lower(?)) > 0 OR instr(lower(b.title), lower(?)) > 0 OR instr(lower(b.author), lower(?)) > 0)`

	const reviewSQL = `
SELECT 'review' AS kind, '' AS bookmark_id, r.review_id AS review_id, r.book_id, r.chapter_uid,
       COALESCE(NULLIF(r.chapter_name,''), c.title, '') AS chapter_title, '' AS mark_text,
       r.content AS content, r.abstract AS abstract, r.create_time AS create_time,
       b.title, b.author, b.cover, 0 AS starred
FROM reviews r
JOIN books b ON b.book_id = r.book_id
LEFT JOIN chapters c ON c.book_id = r.book_id AND c.chapter_uid = r.chapter_uid
WHERE (TRIM(r.content) != '' OR TRIM(r.abstract) != '')
  AND (instr(lower(r.content), lower(?)) > 0 OR instr(lower(r.abstract), lower(?)) > 0 OR instr(lower(b.title), lower(?)) > 0 OR instr(lower(b.author), lower(?)) > 0)`

	var stmt string
	var args []any
	switch kind {
	case "highlight":
		stmt = highlightSQL
		args = []any{query, query, query}
	case "review":
		stmt = reviewSQL
		args = []any{query, query, query, query}
	default:
		stmt = highlightSQL + " UNION ALL " + reviewSQL
		args = []any{query, query, query, query, query, query, query}
	}
	stmt = `SELECT kind, bookmark_id, review_id, book_id, chapter_uid, chapter_title, mark_text, content, abstract,
 create_time, title, author, cover, starred FROM (` + stmt + `) x ORDER BY x.create_time DESC, x.kind ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.DB.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits, err := scanSearchHits(rows)
	if err != nil {
		return nil, err
	}
	if hits == nil {
		hits = []NoteSearchHit{}
	}
	return hits, nil
}

func scanSearchHits(rows *sql.Rows) ([]NoteSearchHit, error) {
	var out []NoteSearchHit
	for rows.Next() {
		var h NoteSearchHit
		var starred int
		if err := rows.Scan(
			&h.Kind, &h.BookmarkID, &h.ReviewID, &h.BookID, &h.ChapterUID, &h.ChapterTitle,
			&h.MarkText, &h.Content, &h.Abstract, &h.CreateTime,
			&h.Title, &h.Author, &h.Cover, &starred,
		); err != nil {
			return nil, err
		}
		h.Starred = starred != 0
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) ListStarredHighlights(limit int) ([]RandomHighlight, error) {
	if limit <= 0 {
		limit = 80
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.DB.Query(`
SELECT h.bookmark_id, h.book_id, h.chapter_uid, h.mark_text, h.create_time, h.range, h.color_style,
       b.title, b.author, b.cover, 1,
       COALESCE(c.title, '')
FROM starred_highlights st
JOIN highlights h ON h.bookmark_id = st.bookmark_id
JOIN books b ON b.book_id = h.book_id
LEFT JOIN chapters c ON c.book_id = h.book_id AND c.chapter_uid = h.chapter_uid
WHERE TRIM(h.mark_text) != ''
ORDER BY st.created_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RandomHighlight
	for rows.Next() {
		var h RandomHighlight
		var starred int
		if err := rows.Scan(
			&h.BookmarkID, &h.BookID, &h.ChapterUID, &h.MarkText, &h.CreateTime, &h.Range, &h.ColorStyle,
			&h.Title, &h.Author, &h.Cover, &starred, &h.Chapter,
		); err != nil {
			return nil, err
		}
		h.Starred = true
		out = append(out, h)
	}
	if out == nil {
		out = []RandomHighlight{}
	}
	return out, rows.Err()
}

func (s *Store) SetHighlightStarred(bookmarkID string, starred bool) error {
	bookmarkID = strings.TrimSpace(bookmarkID)
	if bookmarkID == "" {
		return fmt.Errorf("缺少划线编号")
	}
	if !starred {
		_, err := s.DB.Exec(`DELETE FROM starred_highlights WHERE bookmark_id=?`, bookmarkID)
		return err
	}
	var exists int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM highlights WHERE bookmark_id=?`, bookmarkID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("本地没有这条划线")
	}
	_, err = s.DB.Exec(
		`INSERT OR REPLACE INTO starred_highlights(bookmark_id, created_at) VALUES (?,?)`,
		bookmarkID, time.Now().Unix(),
	)
	return err
}

func (s *Store) ListAllNotebooks() ([]*Book, error) {
	rows, err := s.DB.Query(`SELECT ` + bookCols + ` FROM books WHERE in_notebooks=1 ORDER BY sort DESC, title ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if out == nil {
		out = []*Book{}
	}
	return out, rows.Err()
}

type ExportBook struct {
	Book     *Book
	Chapters []Chapter
	Highlights []Highlight
	Reviews  []Review
}

func (s *Store) LoadExportBook(bookID string) (*ExportBook, error) {
	b, err := s.GetBook(bookID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	chs, err := s.ListChapters(bookID)
	if err != nil {
		return nil, err
	}
	hls, err := s.ListHighlights(bookID)
	if err != nil {
		return nil, err
	}
	revs, err := s.ListReviews(bookID)
	if err != nil {
		return nil, err
	}
	return &ExportBook{Book: b, Chapters: chs, Highlights: hls, Reviews: revs}, nil
}

func (s *Store) LoadExportAll() ([]*ExportBook, error) {
	books, err := s.ListAllNotebooks()
	if err != nil {
		return nil, err
	}
	out := make([]*ExportBook, 0, len(books))
	for _, b := range books {
		pack, err := s.LoadExportBook(b.BookID)
		if err != nil {
			return nil, err
		}
		if pack == nil {
			continue
		}
		out = append(out, pack)
	}
	return out, nil
}

func SafeFilename(title string) string {
	title = strings.TrimSpace(title)
	var b strings.Builder
	for _, r := range title {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteRune('_')
		default:
			if unicode.IsControl(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		s = "notes"
	}
	runes := []rune(s)
	if len(runes) > 60 {
		s = string(runes[:60])
	}
	return s
}

func MarkdownForBooks(packs []*ExportBook) string {
	var b strings.Builder
	b.WriteString("# 纸间笔记导出\n\n")
	b.WriteString("导出时间：" + time.Now().In(Shanghai()).Format("2006-01-02 15:04") + "\n\n")
	for i, pack := range packs {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		writeBookMarkdown(&b, pack)
	}
	return b.String()
}

func MarkdownForBook(pack *ExportBook) string {
	var b strings.Builder
	writeBookMarkdown(&b, pack)
	return b.String()
}

func writeBookMarkdown(b *strings.Builder, pack *ExportBook) {
	book := pack.Book
	title := strings.TrimSpace(book.Title)
	if title == "" {
		title = book.BookID
	}
	b.WriteString("# " + title + "\n\n")
	if strings.TrimSpace(book.Author) != "" {
		b.WriteString("作者：" + strings.TrimSpace(book.Author) + "\n\n")
	}
	type chapPack struct {
		Chapter    Chapter
		Highlights []Highlight
		Reviews    []Review
	}
	order := make([]int64, 0, len(pack.Chapters))
	byUID := map[int64]*chapPack{}
	for _, ch := range pack.Chapters {
		cp := &chapPack{Chapter: ch}
		byUID[ch.ChapterUID] = cp
		order = append(order, ch.ChapterUID)
	}
	orphan := &chapPack{Chapter: Chapter{Title: "未分章", ChapterUID: 0}}
	for _, h := range pack.Highlights {
		if cp := byUID[h.ChapterUID]; cp != nil {
			cp.Highlights = append(cp.Highlights, h)
		} else {
			orphan.Highlights = append(orphan.Highlights, h)
		}
	}
	for _, r := range pack.Reviews {
		if cp := byUID[r.ChapterUID]; cp != nil {
			cp.Reviews = append(cp.Reviews, r)
		} else {
			orphan.Reviews = append(orphan.Reviews, r)
		}
	}
	if len(orphan.Highlights)+len(orphan.Reviews) > 0 {
		order = append(order, 0)
		byUID[0] = orphan
	}
	for _, uid := range order {
		cp := byUID[uid]
		if cp == nil || len(cp.Highlights)+len(cp.Reviews) == 0 {
			continue
		}
		chTitle := strings.TrimSpace(cp.Chapter.Title)
		if chTitle == "" {
			chTitle = "未分章"
		}
		b.WriteString("## " + chTitle + "\n\n")
		used := map[string]bool{}
		for _, h := range cp.Highlights {
			if strings.TrimSpace(h.MarkText) == "" {
				continue
			}
			b.WriteString(blockquote(h.MarkText) + "\n\n")
			for _, r := range cp.Reviews {
				if used[r.ReviewID] || strings.TrimSpace(r.Abstract) == "" {
					continue
				}
				if !textsRelate(r.Abstract, h.MarkText) {
					continue
				}
				used[r.ReviewID] = true
				if strings.TrimSpace(r.Content) != "" {
					b.WriteString(strings.TrimSpace(r.Content) + "\n\n")
				}
			}
		}
		for _, r := range cp.Reviews {
			if used[r.ReviewID] {
				continue
			}
			if strings.TrimSpace(r.Abstract) != "" {
				b.WriteString(blockquote(r.Abstract) + "\n\n")
			}
			if strings.TrimSpace(r.Content) != "" {
				b.WriteString(strings.TrimSpace(r.Content) + "\n\n")
			}
		}
	}
}

func blockquote(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimRight(s, "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

func textsRelate(a, b string) bool {
	x := strings.TrimSpace(a)
	y := strings.TrimSpace(b)
	if x == "" || y == "" {
		return false
	}
	return x == y || strings.Contains(x, y) || strings.Contains(y, x)
}
