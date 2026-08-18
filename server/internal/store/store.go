package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(string(schema)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

type Book struct {
	BookID          string
	Title           string
	Author          string
	Cover           string
	Intro           string
	Category        string
	Publisher       string
	ISBN            string
	ReviewCount     int64
	NoteCount       int64
	BookmarkCount   int64
	ReadingProgress int64
	Sort            int64
	MarkedStatus    int64
	ProgressJSON    string
	InfoJSON        string
	InNotebooks     bool
	IsOnShelf       bool
	IsTop           bool
	FinishReading   bool
	Secret          bool
	ReadUpdateTime  int64
	NotesSyncedAt   int64
	UpdatedAt       int64
}

type Chapter struct {
	BookID     string
	ChapterUID int64
	Title      string
	ChapterIdx int64
}

type Highlight struct {
	BookmarkID string
	BookID     string
	ChapterUID int64
	MarkText   string
	CreateTime int64
	Range      string
	ColorStyle string
}

type RandomHighlight struct {
	Highlight
	Title  string
	Author string
	Cover  string
}

type Review struct {
	ReviewID    string
	BookID      string
	ChapterUID  int64
	ChapterName string
	Content     string
	Abstract    string
	CreateTime  int64
	Star        string
	Range       string
}

func scanBook(sc interface{ Scan(dest ...any) error }) (*Book, error) {
	var b Book
	var inNB, onShelf, isTop, finish, secret int
	err := sc.Scan(
		&b.BookID, &b.Title, &b.Author, &b.Cover, &b.Intro, &b.Category, &b.Publisher, &b.ISBN,
		&b.ReviewCount, &b.NoteCount, &b.BookmarkCount, &b.ReadingProgress, &b.Sort, &b.MarkedStatus,
		&b.ProgressJSON, &b.InfoJSON, &inNB, &onShelf, &isTop, &finish, &secret,
		&b.ReadUpdateTime, &b.NotesSyncedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	b.InNotebooks = inNB != 0
	b.IsOnShelf = onShelf != 0
	b.IsTop = isTop != 0
	b.FinishReading = finish != 0
	b.Secret = secret != 0
	return &b, nil
}

const bookCols = `book_id, title, author, cover, intro, category, publisher, isbn,
	review_count, note_count, bookmark_count, reading_progress, sort, marked_status,
	progress_json, info_json, in_notebooks, is_on_shelf, is_top, finish_reading, secret,
	read_update_time, notes_synced_at, updated_at`

func (s *Store) GetBook(bookID string) (*Book, error) {
	row := s.DB.QueryRow(`SELECT `+bookCols+` FROM books WHERE book_id = ?`, bookID)
	b, err := scanBook(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

func (s *Store) insertBookBase(b *Book) error {
	now := time.Now().Unix()
	if b.UpdatedAt == 0 {
		b.UpdatedAt = now
	}
	_, err := s.DB.Exec(`
INSERT INTO books (`+bookCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(book_id) DO NOTHING
`,
		b.BookID, b.Title, b.Author, b.Cover, b.Intro, b.Category, b.Publisher, b.ISBN,
		b.ReviewCount, b.NoteCount, b.BookmarkCount, b.ReadingProgress, b.Sort, b.MarkedStatus,
		b.ProgressJSON, b.InfoJSON, boolInt(b.InNotebooks), boolInt(b.IsOnShelf), boolInt(b.IsTop),
		boolInt(b.FinishReading), boolInt(b.Secret), b.ReadUpdateTime, b.NotesSyncedAt, b.UpdatedAt,
	)
	return err
}

func (s *Store) UpsertNotebook(b *Book) error {
	if err := s.insertBookBase(b); err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err := s.DB.Exec(`
UPDATE books SET
  title=CASE WHEN ? != '' THEN ? ELSE title END,
  author=CASE WHEN ? != '' THEN ? ELSE author END,
  cover=CASE WHEN ? != '' THEN ? ELSE cover END,
  review_count=?, note_count=?, bookmark_count=?, reading_progress=?, sort=?, marked_status=?,
  in_notebooks=1, updated_at=?
WHERE book_id=?`,
		b.Title, b.Title, b.Author, b.Author, b.Cover, b.Cover,
		b.ReviewCount, b.NoteCount, b.BookmarkCount, b.ReadingProgress, b.Sort, b.MarkedStatus,
		now, b.BookID,
	)
	return err
}

func (s *Store) UpsertShelf(b *Book) error {
	if err := s.insertBookBase(b); err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err := s.DB.Exec(`
UPDATE books SET
  title=CASE WHEN ? != '' THEN ? ELSE title END,
  author=CASE WHEN ? != '' THEN ? ELSE author END,
  cover=CASE WHEN ? != '' THEN ? ELSE cover END,
  category=CASE WHEN ? != '' THEN ? ELSE category END,
  is_on_shelf=1, is_top=?, finish_reading=?, secret=?,
  read_update_time=?, updated_at=?
WHERE book_id=?`,
		b.Title, b.Title, b.Author, b.Author, b.Cover, b.Cover, b.Category, b.Category,
		boolInt(b.IsTop), boolInt(b.FinishReading), boolInt(b.Secret),
		b.ReadUpdateTime, now, b.BookID,
	)
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s *Store) MergeBookFields(bookID string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	cols := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+1)
	for k, v := range fields {
		cols = append(cols, k+"=?")
		args = append(args, v)
	}
	cols = append(cols, "updated_at=?")
	args = append(args, time.Now().Unix(), bookID)
	_, err := s.DB.Exec(`UPDATE books SET `+strings.Join(cols, ", ")+` WHERE book_id=?`, args...)
	return err
}

func (s *Store) ClearNotebooksExcept(ids []string) error {
	now := time.Now().Unix()
	if len(ids) == 0 {
		_, err := s.DB.Exec(`UPDATE books SET in_notebooks=0, updated_at=?`, now)
		return err
	}
	args := append([]any{now}, idsToAny(ids)...)
	_, err := s.DB.Exec(`UPDATE books SET in_notebooks=0, updated_at=? WHERE book_id NOT IN (`+placeholders(len(ids))+`)`, args...)
	return err
}

func (s *Store) SetOnShelf(ids []string) error {
	now := time.Now().Unix()
	if _, err := s.DB.Exec(`UPDATE books SET is_on_shelf=0, updated_at=?`, now); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	ph := placeholders(len(ids))
	args := append([]any{now}, idsToAny(ids)...)
	_, err := s.DB.Exec(`UPDATE books SET is_on_shelf=1, updated_at=? WHERE book_id IN (`+ph+`)`, args...)
	return err
}

func idsToAny(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

func (s *Store) ListNotebooks(count int, lastSort int64) (books []*Book, totalBooks int, totalNotes int, hasMore bool, err error) {
	if count <= 0 {
		count = 40
	}
	err = s.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(note_count+review_count+bookmark_count),0) FROM books WHERE in_notebooks=1`).Scan(&totalBooks, &totalNotes)
	if err != nil {
		return nil, 0, 0, false, err
	}
	q := `SELECT ` + bookCols + ` FROM books WHERE in_notebooks=1`
	args := []any{}
	if lastSort > 0 {
		q += ` AND sort < ?`
		args = append(args, lastSort)
	}
	q += ` ORDER BY sort DESC LIMIT ?`
	args = append(args, count+1)
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, 0, 0, false, err
	}
	defer rows.Close()
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, 0, false, err
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, false, err
	}
	if len(books) > count {
		hasMore = true
		books = books[:count]
	}
	return books, totalBooks, totalNotes, hasMore, nil
}

func (s *Store) ListShelf() ([]*Book, error) {
	rows, err := s.DB.Query(`SELECT ` + bookCols + ` FROM books WHERE is_on_shelf=1 ORDER BY is_top DESC, read_update_time DESC, title ASC`)
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
	return out, rows.Err()
}

func (s *Store) ReplaceNotes(bookID string, chapters []Chapter, highlights []Highlight, reviews []Review) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM chapters WHERE book_id=?`, bookID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM highlights WHERE book_id=?`, bookID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM reviews WHERE book_id=?`, bookID); err != nil {
		return err
	}
	for _, ch := range chapters {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO chapters(book_id, chapter_uid, title, chapter_idx) VALUES (?,?,?,?)`,
			bookID, ch.ChapterUID, ch.Title, ch.ChapterIdx); err != nil {
			return err
		}
	}
	for _, h := range highlights {
		if h.BookmarkID == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO highlights(bookmark_id, book_id, chapter_uid, mark_text, create_time, range, color_style) VALUES (?,?,?,?,?,?,?)`,
			h.BookmarkID, bookID, h.ChapterUID, h.MarkText, h.CreateTime, h.Range, h.ColorStyle); err != nil {
			return err
		}
	}
	for _, r := range reviews {
		if r.ReviewID == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO reviews(review_id, book_id, chapter_uid, chapter_name, content, abstract, create_time, star, range) VALUES (?,?,?,?,?,?,?,?,?)`,
			r.ReviewID, bookID, r.ChapterUID, r.ChapterName, r.Content, r.Abstract, r.CreateTime, r.Star, r.Range); err != nil {
			return err
		}
	}
	now := time.Now().Unix()
	if _, err := tx.Exec(`UPDATE books SET notes_synced_at=?, updated_at=? WHERE book_id=?`, now, now, bookID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListChapters(bookID string) ([]Chapter, error) {
	rows, err := s.DB.Query(`SELECT book_id, chapter_uid, title, chapter_idx FROM chapters WHERE book_id=? ORDER BY chapter_idx, chapter_uid`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chapter
	for rows.Next() {
		var c Chapter
		if err := rows.Scan(&c.BookID, &c.ChapterUID, &c.Title, &c.ChapterIdx); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) RandomHighlights(limit int) ([]RandomHighlight, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 5 {
		limit = 5
	}
	rows, err := s.DB.Query(`
SELECT h.bookmark_id, h.book_id, h.chapter_uid, h.mark_text, h.create_time, h.range, h.color_style,
       b.title, b.author, b.cover
FROM highlights h
JOIN books b ON b.book_id = h.book_id
WHERE TRIM(h.mark_text) != ''
ORDER BY RANDOM()
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RandomHighlight
	for rows.Next() {
		var h RandomHighlight
		if err := rows.Scan(
			&h.BookmarkID, &h.BookID, &h.ChapterUID, &h.MarkText, &h.CreateTime, &h.Range, &h.ColorStyle,
			&h.Title, &h.Author, &h.Cover,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if out == nil {
		out = []RandomHighlight{}
	}
	return out, rows.Err()
}

func (s *Store) ListHighlights(bookID string) ([]Highlight, error) {
	rows, err := s.DB.Query(`SELECT bookmark_id, book_id, chapter_uid, mark_text, create_time, range, color_style FROM highlights WHERE book_id=?`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Highlight
	for rows.Next() {
		var h Highlight
		if err := rows.Scan(&h.BookmarkID, &h.BookID, &h.ChapterUID, &h.MarkText, &h.CreateTime, &h.Range, &h.ColorStyle); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) ListReviews(bookID string) ([]Review, error) {
	rows, err := s.DB.Query(`SELECT review_id, book_id, chapter_uid, chapter_name, content, abstract, create_time, star, range FROM reviews WHERE book_id=?`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Review
	for rows.Next() {
		var r Review
		if err := rows.Scan(&r.ReviewID, &r.BookID, &r.ChapterUID, &r.ChapterName, &r.Content, &r.Abstract, &r.CreateTime, &r.Star, &r.Range); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) PutStats(mode, payload string) error {
	_, err := s.DB.Exec(`INSERT INTO read_stats(mode, payload, fetched_at) VALUES (?,?,?)
ON CONFLICT(mode) DO UPDATE SET payload=excluded.payload, fetched_at=excluded.fetched_at`,
		mode, payload, time.Now().Unix())
	return err
}

func (s *Store) GetStats(mode string) (payload string, fetchedAt int64, err error) {
	err = s.DB.QueryRow(`SELECT payload, fetched_at FROM read_stats WHERE mode=?`, mode).Scan(&payload, &fetchedAt)
	if err == sql.ErrNoRows {
		return "", 0, nil
	}
	return payload, fetchedAt, err
}

func (s *Store) Meta(key string) (string, error) {
	var v string
	err := s.DB.QueryRow(`SELECT value FROM sync_meta WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) SetMeta(key, value string) error {
	_, err := s.DB.Exec(`INSERT INTO sync_meta(key, value) VALUES (?,?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (s *Store) HasAnyBooks() (bool, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&n)
	return n > 0, err
}
