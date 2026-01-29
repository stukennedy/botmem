package memory

import (
	"database/sql"
	"fmt"
	"time"
)

type Summary struct {
	ID        int64     `json:"id"`
	Level     int       `json:"level"`
	Content   string    `json:"content"`
	SourceIDs string    `json:"source_ids,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type SummaryStore struct {
	db *sql.DB
}

func NewSummaryStore(db *sql.DB) *SummaryStore {
	return &SummaryStore{db: db}
}

func (s *SummaryStore) Add(level int, content, sourceIDs string) (*Summary, error) {
	res, err := s.db.Exec(
		`INSERT INTO conversation_summaries (level, content, source_ids) VALUES (?, ?, ?)`,
		level, content, sourceIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("add summary: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *SummaryStore) GetByID(id int64) (*Summary, error) {
	sm := &Summary{}
	err := s.db.QueryRow(
		`SELECT id, level, content, source_ids, created_at FROM conversation_summaries WHERE id = ?`, id,
	).Scan(&sm.ID, &sm.Level, &sm.Content, &sm.SourceIDs, &sm.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get summary %d: %w", id, err)
	}
	return sm, nil
}

func (s *SummaryStore) List(level int, limit int) ([]*Summary, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, level, content, source_ids, created_at FROM conversation_summaries
		WHERE level = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		level, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*Summary
	for rows.Next() {
		sm := &Summary{}
		if err := rows.Scan(&sm.ID, &sm.Level, &sm.Content, &sm.SourceIDs, &sm.CreatedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, sm)
	}
	return summaries, rows.Err()
}

// CountAtLevel returns how many summaries exist at a given level.
func (s *SummaryStore) CountAtLevel(level int) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM conversation_summaries WHERE level = ?`, level,
	).Scan(&count)
	return count, err
}
