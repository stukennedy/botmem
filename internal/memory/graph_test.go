package memory

import (
	"path/filepath"
	"testing"

	"github.com/stukennedy/botmem/internal/db"
)

func testGraphStore(t *testing.T) *GraphStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewGraphStore(database)
}

func TestEnsureEntity(t *testing.T) {
	store := testGraphStore(t)
	id1, err := store.EnsureEntity("Stuart", "person")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if id1 == 0 {
		t.Error("expected non-zero ID")
	}

	// Ensure same entity returns same ID
	id2, err := store.EnsureEntity("Stuart", "person")
	if err != nil {
		t.Fatalf("ensure again: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected same ID, got %d and %d", id1, id2)
	}
}

func TestAddRelation(t *testing.T) {
	store := testGraphStore(t)
	if err := store.AddRelation("Stuart", "works_on", "Moltbot", ""); err != nil {
		t.Fatalf("add: %v", err)
	}

	rels, err := store.QueryEntity("Stuart")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(rels))
	}
	if rels[0].Subject != "Stuart" || rels[0].Predicate != "works_on" || rels[0].Object != "Moltbot" {
		t.Errorf("unexpected relation: %+v", rels[0])
	}
}

func TestAddRelation_Duplicate(t *testing.T) {
	store := testGraphStore(t)
	store.AddRelation("A", "knows", "B", "")
	// Duplicate should not error (INSERT OR IGNORE)
	if err := store.AddRelation("A", "knows", "B", ""); err != nil {
		t.Errorf("unexpected error on duplicate: %v", err)
	}

	rels, _ := store.QueryEntity("A")
	if len(rels) != 1 {
		t.Errorf("expected 1 relation after duplicate insert, got %d", len(rels))
	}
}

func TestQueryEntity_AsSubjectAndObject(t *testing.T) {
	store := testGraphStore(t)
	store.AddRelation("Stuart", "works_on", "Moltbot", "")
	store.AddRelation("Moltbot", "is_a", "Discord bot", "")

	// Query Stuart — should find as subject
	stuartRels, _ := store.QueryEntity("Stuart")
	if len(stuartRels) != 1 {
		t.Errorf("expected 1 for Stuart, got %d", len(stuartRels))
	}

	// Query Moltbot — should find as both subject and object
	moltRels, _ := store.QueryEntity("Moltbot")
	if len(moltRels) != 2 {
		t.Errorf("expected 2 for Moltbot, got %d", len(moltRels))
	}
}

func TestQueryEntity_NotFound(t *testing.T) {
	store := testGraphStore(t)
	rels, err := store.QueryEntity("nobody")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 results, got %d", len(rels))
	}
}

func TestSearchRelations(t *testing.T) {
	store := testGraphStore(t)
	store.AddRelation("Stuart", "works_on", "Moltbot", "")
	store.AddRelation("Stuart", "lives_in", "NZ", "")
	store.AddRelation("Alice", "works_on", "Other", "")

	rels, err := store.SearchRelations("works_on")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rels) != 2 {
		t.Errorf("expected 2 works_on relations, got %d", len(rels))
	}
}

func TestSearchRelations_Partial(t *testing.T) {
	store := testGraphStore(t)
	store.AddRelation("A", "works_on", "B", "")
	store.AddRelation("A", "worked_with", "C", "")

	rels, err := store.SearchRelations("work")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rels) != 2 {
		t.Errorf("expected 2 partial matches, got %d", len(rels))
	}
}

func TestListEntities(t *testing.T) {
	store := testGraphStore(t)
	store.EnsureEntity("Stuart", "person")
	store.EnsureEntity("Moltbot", "project")
	store.EnsureEntity("Go", "language")

	all, err := store.ListEntities("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}

	people, err := store.ListEntities("person")
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 1 {
		t.Errorf("expected 1 person, got %d", len(people))
	}
}

func TestListEntities_Empty(t *testing.T) {
	store := testGraphStore(t)
	entities, err := store.ListEntities("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 0 {
		t.Errorf("expected empty, got %d", len(entities))
	}
}
