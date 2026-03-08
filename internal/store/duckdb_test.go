package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	internalaws "github.com/pfrederiksen/cloudnecromancer/internal/aws"
	"github.com/pfrederiksen/cloudnecromancer/internal/store"
)

func tempDB(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeEvents(ids ...string) []internalaws.RawEvent {
	base := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	events := make([]internalaws.RawEvent, len(ids))
	for i, id := range ids {
		events[i] = internalaws.RawEvent{
			EventID:     id,
			EventTime:   base.Add(time.Duration(i) * time.Minute),
			EventName:   "CreateBucket",
			EventSource: "s3.amazonaws.com",
			Region:      "us-east-1",
			RawJSON:     `{"id":"` + id + `"}`,
		}
	}
	return events
}

func TestInsertAndQuery(t *testing.T) {
	tests := []struct {
		name        string
		events      []internalaws.RawEvent
		wantInserted int
	}{
		{
			name:        "insert two events",
			events:      makeEvents("e1", "e2"),
			wantInserted: 2,
		},
		{
			name:        "insert empty slice",
			events:      nil,
			wantInserted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tempDB(t)
			n, err := s.InsertEvents(tt.events, "123456789012")
			if err != nil {
				t.Fatalf("InsertEvents: %v", err)
			}
			if n != tt.wantInserted {
				t.Errorf("inserted %d, want %d", n, tt.wantInserted)
			}
		})
	}
}

func TestInsertDedup(t *testing.T) {
	s := tempDB(t)

	// Insert first batch
	n1, err := s.InsertEvents(makeEvents("e1", "e2"), "123456789012")
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if n1 != 2 {
		t.Errorf("first insert: got %d, want 2", n1)
	}

	// Insert overlapping batch — e2 is a duplicate
	n2, err := s.InsertEvents(makeEvents("e2", "e3"), "123456789012")
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if n2 != 1 {
		t.Errorf("second insert: got %d, want 1 (e3 only)", n2)
	}

	// Verify total count
	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.EventCount != 3 {
		t.Errorf("total count: got %d, want 3", stats.EventCount)
	}
}

func TestQueryFilters(t *testing.T) {
	s := tempDB(t)

	base := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	events := []internalaws.RawEvent{
		{EventID: "e1", EventTime: base, EventName: "CreateBucket", EventSource: "s3.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
		{EventID: "e2", EventTime: base.Add(time.Minute), EventName: "RunInstances", EventSource: "ec2.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
		{EventID: "e3", EventTime: base.Add(2 * time.Minute), EventName: "CreateTable", EventSource: "dynamodb.amazonaws.com", Region: "eu-west-1", RawJSON: "{}"},
	}
	if _, err := s.InsertEvents(events, "123456789012"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	tests := []struct {
		name      string
		before    time.Time
		services  []string
		regions   []string
		wantCount int
	}{
		{
			name:      "all events",
			before:    base.Add(time.Hour),
			wantCount: 3,
		},
		{
			name:      "filter by service",
			before:    base.Add(time.Hour),
			services:  []string{"s3.amazonaws.com"},
			wantCount: 1,
		},
		{
			name:      "filter by region",
			before:    base.Add(time.Hour),
			regions:   []string{"eu-west-1"},
			wantCount: 1,
		},
		{
			name:      "filter by time",
			before:    base.Add(30 * time.Second),
			wantCount: 1,
		},
		{
			name:      "combined filters",
			before:    base.Add(time.Hour),
			services:  []string{"ec2.amazonaws.com"},
			regions:   []string{"us-east-1"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := s.QueryEvents(tt.before, tt.services, tt.regions)
			if err != nil {
				t.Fatalf("QueryEvents: %v", err)
			}
			if len(results) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestQueryEventsOrdering(t *testing.T) {
	s := tempDB(t)

	base := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	events := []internalaws.RawEvent{
		{EventID: "e3", EventTime: base.Add(2 * time.Minute), EventName: "C", EventSource: "s3.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
		{EventID: "e1", EventTime: base, EventName: "A", EventSource: "s3.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
		{EventID: "e2", EventTime: base.Add(time.Minute), EventName: "B", EventSource: "s3.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
	}
	if _, err := s.InsertEvents(events, "acct"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	results, err := s.QueryEvents(base.Add(time.Hour), nil, nil)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	// Should be ordered by event_time ASC
	if results[0].EventID != "e1" || results[1].EventID != "e2" || results[2].EventID != "e3" {
		t.Errorf("wrong order: %s, %s, %s", results[0].EventID, results[1].EventID, results[2].EventID)
	}
}

func TestStats(t *testing.T) {
	tests := []struct {
		name         string
		events       []internalaws.RawEvent
		wantCount    int
		wantServices int
	}{
		{
			name:         "empty store",
			events:       nil,
			wantCount:    0,
			wantServices: 0,
		},
		{
			name: "with events",
			events: []internalaws.RawEvent{
				{EventID: "e1", EventTime: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), EventName: "X", EventSource: "s3.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
				{EventID: "e2", EventTime: time.Date(2026, 1, 15, 13, 0, 0, 0, time.UTC), EventName: "Y", EventSource: "ec2.amazonaws.com", Region: "us-east-1", RawJSON: "{}"},
			},
			wantCount:    2,
			wantServices: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tempDB(t)
			if len(tt.events) > 0 {
				if _, err := s.InsertEvents(tt.events, "acct"); err != nil {
					t.Fatalf("insert: %v", err)
				}
			}
			stats, err := s.Stats()
			if err != nil {
				t.Fatalf("Stats: %v", err)
			}
			if stats.EventCount != tt.wantCount {
				t.Errorf("count: got %d, want %d", stats.EventCount, tt.wantCount)
			}
			if len(stats.Services) != tt.wantServices {
				t.Errorf("services: got %d, want %d", len(stats.Services), tt.wantServices)
			}
		})
	}
}

func TestNewStore_InvalidPath(t *testing.T) {
	// Attempt to open a store in a non-existent deeply nested directory
	_, err := store.NewStore("/nonexistent/path/that/does/not/exist/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestNewStore_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "new.db")

	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected database file to be created")
	}
}
