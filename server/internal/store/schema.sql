CREATE TABLE IF NOT EXISTS books (
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

CREATE TABLE IF NOT EXISTS chapters (
  book_id TEXT NOT NULL,
  chapter_uid INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  chapter_idx INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (book_id, chapter_uid)
);

CREATE TABLE IF NOT EXISTS highlights (
  bookmark_id TEXT PRIMARY KEY,
  book_id TEXT NOT NULL,
  chapter_uid INTEGER NOT NULL DEFAULT 0,
  mark_text TEXT NOT NULL DEFAULT '',
  create_time INTEGER NOT NULL DEFAULT 0,
  range TEXT NOT NULL DEFAULT '',
  color_style TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS reviews (
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

CREATE TABLE IF NOT EXISTS read_stats (
  mode TEXT PRIMARY KEY,
  payload TEXT NOT NULL,
  fetched_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS app_settings (
  k TEXT PRIMARY KEY,
  v TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_books_notebooks_sort ON books(in_notebooks, sort DESC);
CREATE INDEX IF NOT EXISTS idx_books_shelf ON books(is_on_shelf, is_top DESC, read_update_time DESC);
CREATE INDEX IF NOT EXISTS idx_highlights_book ON highlights(book_id);
CREATE INDEX IF NOT EXISTS idx_reviews_book ON reviews(book_id);
CREATE INDEX IF NOT EXISTS idx_chapters_book ON chapters(book_id);
