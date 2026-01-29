package memory

import (
	"database/sql"
	"fmt"
	"time"
)

type Entity struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	EntityType string    `json:"entity_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type Relation struct {
	ID        int64     `json:"id"`
	Subject   string    `json:"subject"`
	Predicate string    `json:"predicate"`
	Object    string    `json:"object"`
	Metadata  string    `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type GraphStore struct {
	db *sql.DB
}

func NewGraphStore(db *sql.DB) *GraphStore {
	return &GraphStore{db: db}
}

// EnsureEntity creates an entity if it doesn't exist, returns its ID either way.
func (s *GraphStore) EnsureEntity(name, entityType string) (int64, error) {
	// Try insert, ignore conflict
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO entities (name, entity_type) VALUES (?, ?)`,
		name, entityType,
	)
	if err != nil {
		return 0, fmt.Errorf("ensure entity: %w", err)
	}

	var id int64
	err = s.db.QueryRow(`SELECT id FROM entities WHERE name = ?`, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get entity id: %w", err)
	}
	return id, nil
}

// AddRelation adds a subject-predicate-object triplet.
func (s *GraphStore) AddRelation(subject, predicate, object, metadata string) error {
	subID, err := s.EnsureEntity(subject, "")
	if err != nil {
		return err
	}
	objID, err := s.EnsureEntity(object, "")
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT OR IGNORE INTO relations (subject_id, predicate, object_id, metadata) VALUES (?, ?, ?, ?)`,
		subID, predicate, objID, metadata,
	)
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}
	return nil
}

// QueryEntity returns all relations where the given entity is subject or object.
func (s *GraphStore) QueryEntity(name string) ([]*Relation, error) {
	rows, err := s.db.Query(
		`SELECT r.id, s.name, r.predicate, o.name, r.metadata, r.created_at
		FROM relations r
		JOIN entities s ON s.id = r.subject_id
		JOIN entities o ON o.id = r.object_id
		WHERE s.name = ? OR o.name = ?
		ORDER BY r.created_at DESC`,
		name, name,
	)
	if err != nil {
		return nil, fmt.Errorf("query entity: %w", err)
	}
	defer rows.Close()

	var rels []*Relation
	for rows.Next() {
		r := &Relation{}
		if err := rows.Scan(&r.ID, &r.Subject, &r.Predicate, &r.Object, &r.Metadata, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// SearchRelations searches for relations matching a predicate pattern.
func (s *GraphStore) SearchRelations(predicate string) ([]*Relation, error) {
	rows, err := s.db.Query(
		`SELECT r.id, s.name, r.predicate, o.name, r.metadata, r.created_at
		FROM relations r
		JOIN entities s ON s.id = r.subject_id
		JOIN entities o ON o.id = r.object_id
		WHERE r.predicate LIKE ?
		ORDER BY r.created_at DESC`,
		"%"+predicate+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search relations: %w", err)
	}
	defer rows.Close()

	var rels []*Relation
	for rows.Next() {
		r := &Relation{}
		if err := rows.Scan(&r.ID, &r.Subject, &r.Predicate, &r.Object, &r.Metadata, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// ListEntities returns all entities, optionally filtered by type.
func (s *GraphStore) ListEntities(entityType string) ([]*Entity, error) {
	query := `SELECT id, name, entity_type, created_at FROM entities`
	var args []any
	if entityType != "" {
		query += ` WHERE entity_type = ?`
		args = append(args, entityType)
	}
	query += ` ORDER BY name`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		e := &Entity{}
		if err := rows.Scan(&e.ID, &e.Name, &e.EntityType, &e.CreatedAt); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}
