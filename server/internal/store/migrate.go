package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func execScript(db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.Exec(stmt); err != nil {
			preview := stmt
			if len(preview) > 80 {
				preview = preview[:80] + "…"
			}
			return fmt.Errorf("%s: %w", preview, err)
		}
	}
	return nil
}

func splitSQL(script string) []string {
	var out []string
	for _, part := range strings.Split(script, ";") {
		stmt := strings.TrimSpace(part)
		if stmt == "" || strings.HasPrefix(stmt, "--") && !strings.Contains(stmt, "\n") {
			continue
		}
		lines := strings.Split(stmt, "\n")
		var kept []string
		for _, line := range lines {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "--") {
				continue
			}
			kept = append(kept, line)
		}
		stmt = strings.TrimSpace(strings.Join(kept, "\n"))
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func (s *Store) migrate() error {
	if err := s.migrateReadStatsYear(); err != nil {
		return fmt.Errorf("read_stats year: %w", err)
	}
	if err := s.migrateReadStatsPeriod(); err != nil {
		return fmt.Errorf("read_stats period: %w", err)
	}
	return nil
}

func (s *Store) tableColumnCount(table, column string) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&n)
	return n, err
}

func (s *Store) migrateReadStatsYear() error {
	n, err := s.tableColumnCount("read_stats", "year")
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	type statRow struct {
		mode      string
		payload   string
		fetchedAt int64
	}
	var copied []statRow
	rows, err := s.DB.Query(`SELECT mode, payload, fetched_at FROM read_stats`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var r statRow
		if err := rows.Scan(&r.mode, &r.payload, &r.fetchedAt); err != nil {
			_ = rows.Close()
			return err
		}
		copied = append(copied, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DROP TABLE IF EXISTS read_stats_v2`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE TABLE read_stats_v2 (
  mode TEXT NOT NULL,
  year INTEGER NOT NULL DEFAULT 0,
  payload TEXT NOT NULL,
  fetched_at INTEGER NOT NULL,
  PRIMARY KEY (mode, year)
)`); err != nil {
		return err
	}

	now := time.Now()
	for _, r := range copied {
		year := int64(0)
		if r.mode == "annually" {
			year = int64(YearFromPayload(r.payload, time.Unix(r.fetchedAt, 0)))
			if year <= 0 {
				year = int64(CalendarYear(now))
			}
		}
		if _, err := tx.Exec(`INSERT INTO read_stats_v2(mode, year, payload, fetched_at) VALUES (?,?,?,?)
ON CONFLICT(mode, year) DO UPDATE SET payload=excluded.payload, fetched_at=excluded.fetched_at`,
			r.mode, year, r.payload, r.fetchedAt); err != nil {
			return err
		}
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
	if err := rows.Close(); err != nil {
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
