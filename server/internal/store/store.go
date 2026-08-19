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
	st := &Store{DB: db}
	if err := st.migrateReadStatsYear(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate read_stats: %w", err)
	}
	if err := st.migrateReadStatsPeriod(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate read_stats period: %w", err)
	}
	return st, nil
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

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (s *Store) ListNotebooks(count int, lastSort int64, query string) (books []*Book, totalBooks int, totalNotes int, hasMore bool, err error) {
	if count <= 0 {
		count = 40
	}
	where := `in_notebooks=1`
	countArgs := []any{}
	query = strings.TrimSpace(query)
	if query != "" {
		pat := "%" + escapeLike(query) + "%"
		where += ` AND (title LIKE ? ESCAPE '\' OR author LIKE ? ESCAPE '\')`
		countArgs = append(countArgs, pat, pat)
	}
	err = s.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(note_count+review_count+bookmark_count),0) FROM books WHERE `+where, countArgs...).Scan(&totalBooks, &totalNotes)
	if err != nil {
		return nil, 0, 0, false, err
	}
	q := `SELECT ` + bookCols + ` FROM books WHERE ` + where
	args := append([]any{}, countArgs...)
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

func (s *Store) migrateReadStatsYear() error {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('read_stats') WHERE name='year'`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`CREATE TABLE read_stats_v2 (
  mode TEXT NOT NULL,
  year INTEGER NOT NULL DEFAULT 0,
  payload TEXT NOT NULL,
  fetched_at INTEGER NOT NULL,
  PRIMARY KEY (mode, year)
)`); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT mode, payload, fetched_at FROM read_stats`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now()
	for rows.Next() {
		var mode, payload string
		var fetchedAt int64
		if err := rows.Scan(&mode, &payload, &fetchedAt); err != nil {
			return err
		}
		year := int64(0)
		if mode == "annually" {
			year = int64(YearFromPayload(payload, time.Unix(fetchedAt, 0)))
			if year <= 0 {
				year = int64(CalendarYear(now))
			}
		}
		if _, err := tx.Exec(`INSERT INTO read_stats_v2(mode, year, payload, fetched_at) VALUES (?,?,?,?)
ON CONFLICT(mode, year) DO UPDATE SET payload=excluded.payload, fetched_at=excluded.fetched_at`,
			mode, year, payload, fetchedAt); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE read_stats`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE read_stats_v2 RENAME TO read_stats`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) migrateReadStatsPeriod() error {
	rows, err := s.DB.Query(`SELECT mode, year, payload, fetched_at FROM read_stats WHERE mode IN ('weekly','monthly') AND year=0`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		mode      string
		payload   string
		fetchedAt int64
	}
	var pending []row
	for rows.Next() {
		var r row
		var year int64
		if err := rows.Scan(&r.mode, &year, &r.payload, &r.fetchedAt); err != nil {
			return err
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now()
	for _, r := range pending {
		key := PeriodKeyFromPayload(r.mode, r.payload, time.Unix(r.fetchedAt, 0))
		if key <= 0 {
			key = PeriodKeyFromPayload(r.mode, r.payload, now)
		}
		if _, err := s.DB.Exec(`INSERT INTO read_stats(mode, year, payload, fetched_at) VALUES (?,?,?,?)
ON CONFLICT(mode, year) DO UPDATE SET
  payload=CASE WHEN excluded.fetched_at>=read_stats.fetched_at THEN excluded.payload ELSE read_stats.payload END,
  fetched_at=CASE WHEN excluded.fetched_at>=read_stats.fetched_at THEN excluded.fetched_at ELSE read_stats.fetched_at END`,
			r.mode, key, r.payload, r.fetchedAt); err != nil {
			return err
		}
		if _, err := s.DB.Exec(`DELETE FROM read_stats WHERE mode=? AND year=0`, r.mode); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) PutStats(mode, payload string) error {
	return s.PutStatsYear(mode, PeriodKeyFromPayload(mode, payload, time.Now()), payload)
}

func (s *Store) PutStatsYear(mode string, year int64, payload string) error {
	_, err := s.DB.Exec(`INSERT INTO read_stats(mode, year, payload, fetched_at) VALUES (?,?,?,?)
ON CONFLICT(mode, year) DO UPDATE SET payload=excluded.payload, fetched_at=excluded.fetched_at`,
		mode, year, payload, time.Now().Unix())
	return err
}

func (s *Store) GetStats(mode string) (payload string, fetchedAt int64, err error) {
	now := time.Now()
	var key int64
	switch mode {
	case "annually":
		key = int64(CalendarYear(now))
	case "monthly":
		key = MonthKey(now)
	case "weekly":
		key = WeekKey(now)
	default:
		key = 0
	}
	payload, fetchedAt, err = s.GetStatsYear(mode, key)
	if err != nil || payload != "" {
		return payload, fetchedAt, err
	}
	if mode == "overall" {
		return "", 0, nil
	}
	err = s.DB.QueryRow(`SELECT payload, fetched_at FROM read_stats WHERE mode=? AND year>0 ORDER BY year DESC LIMIT 1`, mode).Scan(&payload, &fetchedAt)
	if err == sql.ErrNoRows {
		return "", 0, nil
	}
	return payload, fetchedAt, err
}

func (s *Store) GetStatsYear(mode string, year int64) (payload string, fetchedAt int64, err error) {
	err = s.DB.QueryRow(`SELECT payload, fetched_at FROM read_stats WHERE mode=? AND year=?`, mode, year).Scan(&payload, &fetchedAt)
	if err == sql.ErrNoRows {
		return "", 0, nil
	}
	return payload, fetchedAt, err
}

func (s *Store) ListAnnualYears() ([]int, error) {
	rows, err := s.DB.Query(`SELECT year FROM read_stats WHERE mode='annually' AND year>0 ORDER BY year DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, rows.Err()
}

func (s *Store) NoteTimeRange() (minTs, maxTs int64, err error) {
	err = s.DB.QueryRow(`SELECT COALESCE(MIN(ts),0), COALESCE(MAX(ts),0) FROM (
  SELECT create_time AS ts FROM highlights WHERE create_time>0
  UNION ALL
  SELECT create_time AS ts FROM reviews WHERE create_time>0
)`).Scan(&minTs, &maxTs)
	return minTs, maxTs, err
}

type YearNote struct {
	Kind       string
	ID         string
	BookID     string
	CreateTime int64
	Title      string
	Author     string
	Cover      string
	InfoJSON   string
}

func (s *Store) ListYearNotes(fromTs, toTs int64) ([]YearNote, error) {
	rows, err := s.DB.Query(`
SELECT 'highlight', h.bookmark_id, h.book_id, h.create_time, b.title, b.author, b.cover, b.info_json
FROM highlights h JOIN books b ON b.book_id=h.book_id
WHERE h.create_time>=? AND h.create_time<?
UNION ALL
SELECT 'review', r.review_id, r.book_id, r.create_time, b.title, b.author, b.cover, b.info_json
FROM reviews r JOIN books b ON b.book_id=r.book_id
WHERE r.create_time>=? AND r.create_time<?
ORDER BY 4 ASC`, fromTs, toTs, fromTs, toTs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []YearNote
	for rows.Next() {
		var n YearNote
		if err := rows.Scan(&n.Kind, &n.ID, &n.BookID, &n.CreateTime, &n.Title, &n.Author, &n.Cover, &n.InfoJSON); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
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

func (s *Store) GetSetting(key string) (string, error) {
	var v string
	err := s.DB.QueryRow(`SELECT v FROM app_settings WHERE k=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec(`INSERT INTO app_settings(k, v) VALUES (?,?)
ON CONFLICT(k) DO UPDATE SET v=excluded.v`, key, value)
	return err
}

type AppSettings struct {
	APIKeyCipher string
	SkillVersion string
	GatewayURL   string
	SyncInterval string
	SiteTitle    string
	Theme        string
	ColorScheme  string
}

func (s *Store) LoadAppSettings() (AppSettings, error) {
	var out AppSettings
	rows, err := s.DB.Query(`SELECT k, v FROM app_settings`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "api_key":
			out.APIKeyCipher = v
		case "skill_version":
			out.SkillVersion = v
		case "gateway_url":
			out.GatewayURL = v
		case "sync_interval":
			out.SyncInterval = v
		case "site_title":
			out.SiteTitle = v
		case "theme":
			out.Theme = v
		case "color_scheme":
			out.ColorScheme = v
		}
	}
	return out, rows.Err()
}
