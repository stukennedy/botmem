package memory

import (
	"path/filepath"
	"testing"

	"github.com/stukennedy/botmem/internal/db"
)

func testSummaryStore(t *testing.T) *SummaryStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSummaryStore(database)
}

func TestSummaryAdd(t *testing.T) {
	store := testSummaryStore(t)
	s, err := store.Add(0, "Discussed memory architecture", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if s.Level != 0 || s.Content != "Discussed memory architecture" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestSummaryAdd_WithSourceIDs(t *testing.T) {
	store := testSummaryStore(t)
	s, err := store.Add(1, "Meta-summary", "1,2,3")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if s.SourceIDs != "1,2,3" {
		t.Errorf("expected source_ids '1,2,3', got %q", s.SourceIDs)
	}
}

func TestSummaryList_ByLevel(t *testing.T) {
	store := testSummaryStore(t)
	store.Add(0, "L0 summary 1", "")
	store.Add(0, "L0 summary 2", "")
	store.Add(1, "L1 summary", "")

	l0, err := store.List(0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(l0) != 2 {
		t.Errorf("expected 2 L0 summaries, got %d", len(l0))
	}

	l1, err := store.List(1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(l1) != 1 {
		t.Errorf("expected 1 L1 summary, got %d", len(l1))
	}
}

func TestSummaryList_DefaultLimit(t *testing.T) {
	store := testSummaryStore(t)
	for i := 0; i < 30; i++ {
		store.Add(0, "summary", "")
	}

	results, err := store.List(0, 0) // 0 defaults to 20
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 20 {
		t.Errorf("expected max 20 with default limit, got %d", len(results))
	}
}

func TestSummaryList_OrderedByRecent(t *testing.T) {
	store := testSummaryStore(t)
	store.Add(0, "first", "")
	store.Add(0, "second", "")

	results, _ := store.List(0, 20)
	// DESC order by created_at, then by id — higher ID = more recent
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
	if results[0].ID < results[1].ID {
		t.Errorf("expected most recent (higher ID) first, got IDs %d, %d", results[0].ID, results[1].ID)
	}
}

func TestSummaryCountAtLevel(t *testing.T) {
	store := testSummaryStore(t)
	store.Add(0, "a", "")
	store.Add(0, "b", "")
	store.Add(1, "c", "")

	count, err := store.CountAtLevel(0)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	count, _ = store.CountAtLevel(2)
	if count != 0 {
		t.Errorf("expected 0 at level 2, got %d", count)
	}
}
