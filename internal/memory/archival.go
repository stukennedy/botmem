package memory

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ArchivalEntry struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content"`
	Tags      string    `json:"tags"`
	Embedding []byte    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type ArchivalStore struct {
	db *sql.DB
}

func NewArchivalStore(db *sql.DB) *ArchivalStore {
	return &ArchivalStore{db: db}
}

func (s *ArchivalStore) Add(content string, tags []string, embedding []byte) (*ArchivalEntry, error) {
	tagStr := strings.Join(tags, ",")
	res, err := s.db.Exec(
		`INSERT INTO archival (content, tags, embedding) VALUES (?, ?, ?)`,
		content, tagStr, embedding,
	)
	if err != nil {
		return nil, fmt.Errorf("add archival: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *ArchivalStore) GetByID(id int64) (*ArchivalEntry, error) {
	e := &ArchivalEntry{}
	err := s.db.QueryRow(
		`SELECT id, content, tags, embedding, created_at FROM archival WHERE id = ?`, id,
	).Scan(&e.ID, &e.Content, &e.Tags, &e.Embedding, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get archival %d: %w", id, err)
	}
	return e, nil
}

func (s *ArchivalStore) Search(query string, limit int) ([]*ArchivalEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(
		`SELECT a.id, a.content, a.tags, a.created_at
		FROM archival_fts f
		JOIN archival a ON a.id = f.rowid
		WHERE archival_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search archival: %w", err)
	}
	defer rows.Close()

	var entries []*ArchivalEntry
	for rows.Next() {
		e := &ArchivalEntry{}
		if err := rows.Scan(&e.ID, &e.Content, &e.Tags, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *ArchivalStore) List(tag string, limit int) ([]*ArchivalEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, content, tags, created_at FROM archival`
	var args []any
	if tag != "" {
		query += ` WHERE tags LIKE ?`
		args = append(args, "%"+tag+"%")
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list archival: %w", err)
	}
	defer rows.Close()

	var entries []*ArchivalEntry
	for rows.Next() {
		e := &ArchivalEntry{}
		if err := rows.Scan(&e.ID, &e.Content, &e.Tags, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *ArchivalStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM archival WHERE id = ?`, id)
	return err
}

// SearchWithEmbedding retrieves all entries with embeddings for cosine similarity comparison.
// The actual similarity computation happens in Go.
func (s *ArchivalStore) AllWithEmbeddings() ([]*ArchivalEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, content, tags, embedding, created_at FROM archival WHERE embedding IS NOT NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("list embeddings: %w", err)
	}
	defer rows.Close()

	var entries []*ArchivalEntry
	for rows.Next() {
		e := &ArchivalEntry{}
		if err := rows.Scan(&e.ID, &e.Content, &e.Tags, &e.Embedding, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
