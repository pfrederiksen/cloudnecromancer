package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"

	internalaws "github.com/pfrederiksen/cloudnecromancer/internal/aws"
)

// StoredEvent represents an event row read back from the store.
type StoredEvent struct {
	EventID      string
	EventTime    time.Time
	EventName    string
	EventSource  string
	ResourceType string
	ResourceID   string
	AccountID    string
	Region       string
	RawJSON      string
}

// StoreStats contains aggregate statistics about the event store.
type StoreStats struct {
	EventCount int
	MinTime    time.Time
	MaxTime    time.Time
	Services   []string
}

// Store wraps a DuckDB connection for event caching.
type Store struct {
	db *sql.DB
}

const createTableSQL = `
CREATE TABLE IF NOT EXISTS events (
	event_id      TEXT PRIMARY KEY,
	event_time    TIMESTAMP,
	event_name    TEXT,
	event_source  TEXT,
	resource_type TEXT,
	resource_id   TEXT,
	account_id    TEXT,
	region        TEXT,
	raw_json      TEXT
);
`

// NewStore opens (or creates) a DuckDB database at dbPath and ensures the
// events table exists.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", dbPath, err)
	}
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}
	return &Store{db: db}, nil
}

// InsertEvents bulk-inserts events, ignoring duplicates by event_id.
// It returns the number of newly inserted rows.
func (s *Store) InsertEvents(events []internalaws.RawEvent, accountID string) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO events
			(event_id, event_time, event_name, event_source, resource_type, resource_id, account_id, region, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, ev := range events {
		res, err := stmt.Exec(
			ev.EventID,
			ev.EventTime,
			ev.EventName,
			ev.EventSource,
			"", // resource_type – populated later by parser
			"", // resource_id  – populated later by parser
			accountID,
			ev.Region,
			ev.RawJSON,
		)
		if err != nil {
			return inserted, fmt.Errorf("insert event %s: %w", ev.EventID, err)
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// QueryEvents returns events matching optional filters, ordered by time ASC.
// If services is empty, all services are included. Same for regions.
// The before parameter limits events to those occurring before the given time.
func (s *Store) QueryEvents(before time.Time, services []string, regions []string) ([]StoredEvent, error) {
	query := "SELECT event_id, event_time, event_name, event_source, resource_type, resource_id, account_id, region, raw_json FROM events WHERE event_time <= ?"
	args := []any{before}

	if len(services) > 0 {
		placeholders := make([]string, len(services))
		for i, svc := range services {
			placeholders[i] = "?"
			args = append(args, svc)
		}
		query += " AND event_source IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(regions) > 0 {
		placeholders := make([]string, len(regions))
		for i, r := range regions {
			placeholders[i] = "?"
			args = append(args, r)
		}
		query += " AND region IN (" + strings.Join(placeholders, ",") + ")"
	}

	query += " ORDER BY event_time ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var result []StoredEvent
	for rows.Next() {
		var ev StoredEvent
		if err := rows.Scan(
			&ev.EventID, &ev.EventTime, &ev.EventName, &ev.EventSource,
			&ev.ResourceType, &ev.ResourceID, &ev.AccountID, &ev.Region, &ev.RawJSON,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, ev)
	}
	return result, rows.Err()
}

// Stats returns aggregate statistics about the event store.
func (s *Store) Stats() (*StoreStats, error) {
	stats := &StoreStats{}

	row := s.db.QueryRow("SELECT COUNT(*) FROM events")
	if err := row.Scan(&stats.EventCount); err != nil {
		return nil, fmt.Errorf("count: %w", err)
	}

	if stats.EventCount == 0 {
		return stats, nil
	}

	row = s.db.QueryRow("SELECT MIN(event_time), MAX(event_time) FROM events")
	if err := row.Scan(&stats.MinTime, &stats.MaxTime); err != nil {
		return nil, fmt.Errorf("time range: %w", err)
	}

	rows, err := s.db.Query("SELECT DISTINCT event_source FROM events ORDER BY event_source")
	if err != nil {
		return nil, fmt.Errorf("services: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var svc string
		if err := rows.Scan(&svc); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		stats.Services = append(stats.Services, svc)
	}
	return stats, rows.Err()
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
