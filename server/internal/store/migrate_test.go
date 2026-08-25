package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const legacySchema = `
CREATE TABLE books (
  book_id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL DEFAULT '',
  cover TEXT NOT NULL DEFAULT '',
  intro TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  publisher TEXT NOT NULL DEFAULT '',
  isbn TEXT NOT NULL DEFAULT '',
  review_count INTEGER NOT NULL DEFAULT 0,
  note_count INTEGER NOT NULL DEFAULT 0,
  bookmark_count INTEGER NOT NULL DEFAULT 0,
  reading_progress INTEGER NOT NULL DEFAULT 0,
  sort INTEGER NOT NULL DEFAULT 0,
  marked_status INTEGER NOT NULL DEFAULT 0,
  progress_json TEXT NOT NULL DEFAULT '',
  info_json TEXT NOT NULL DEFAULT '',
  in_notebooks INTEGER NOT NULL DEFAULT 0,
  is_on_shelf INTEGER NOT NULL DEFAULT 0,
  is_top INTEGER NOT NULL DEFAULT 0,
  finish_reading INTEGER NOT NULL DEFAULT 0,
  secret INTEGER NOT NULL DEFAULT 0,
  read_update_time INTEGER NOT NULL DEFAULT 0,
  notes_synced_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE highlights (
  bookmark_id TEXT PRIMARY KEY,
  book_id TEXT NOT NULL,
  chapter_uid INTEGER NOT NULL DEFAULT 0,
  mark_text TEXT NOT NULL DEFAULT '',
  create_time INTEGER NOT NULL DEFAULT 0,
  range TEXT NOT NULL DEFAULT '',
  color_style TEXT NOT NULL DEFAULT ''
);
CREATE TABLE reviews (
  review_id TEXT PRIMARY KEY,
  book_id TEXT NOT NULL,
  chapter_uid INTEGER NOT NULL DEFAULT 0,
  chapter_name TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  abstract TEXT NOT NULL DEFAULT '',
  create_time INTEGER NOT NULL DEFAULT 0,
  star TEXT NOT NULL DEFAULT '',
  range TEXT NOT NULL DEFAULT ''
);
CREATE TABLE chapters (
  book_id TEXT NOT NULL,
  chapter_uid INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  chapter_idx INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (book_id, chapter_uid)
);
CREATE TABLE read_stats (
  mode TEXT PRIMARY KEY,
  payload TEXT NOT NULL,
  fetched_at INTEGER NOT NULL
);
CREATE TABLE sync_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

func TestOpenMigratesLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books(book_id, title, in_notebooks, sort) VALUES ('b1','旧书',1,9)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO highlights(bookmark_id, book_id, mark_text, create_time) VALUES ('h1','b1','一句旧划线',11)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO read_stats(mode, payload, fetched_at) VALUES ('overall','{"totalReadingTime":1}',100)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := Open(path)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	book, err := st.GetBook("b1")
	if err != nil || book == nil || book.Title != "旧书" {
		t.Fatalf("book after migrate: %v %+v", err, book)
	}
	if err := st.SetHighlightStarred("h1", true); err != nil {
		t.Fatalf("starred_highlights missing: %v", err)
	}
	if err := st.SetSetting("site_title", "测试"); err != nil {
		t.Fatalf("app_settings missing: %v", err)
	}
	payload, _, err := st.GetStatsYear("overall", 0)
	if err != nil || payload == "" {
		t.Fatalf("read_stats year migrate: %v %q", err, payload)
	}
}

func TestSplitSQLSkipsComments(t *testing.T) {
	stmts := splitSQL("-- comment\nCREATE TABLE a (id INT);\nCREATE TABLE b (id INT);")
	if len(stmts) != 2 {
		t.Fatalf("got %#v", stmts)
	}
}
