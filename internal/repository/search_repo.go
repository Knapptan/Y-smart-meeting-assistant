package repository

import (
	"context"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// SearchRepo — реализация domain.SearchRepository.
type SearchRepo struct {
	db *DB
}

// NewSearchRepo создаёт репозиторий поиска.
func NewSearchRepo(db *DB) *SearchRepo {
	return &SearchRepo{db: db}
}

// FindByKeyword ищет встречи пользователя, содержащие keyword
// в транскрипции или выжимке (ILIKE, регистронезависимо).
func (r *SearchRepo) FindByKeyword(ctx context.Context, userID int64, keyword string) ([]*domain.Meeting, error) {
	const q = `
		SELECT m.id, m.user_id, m.title, m.status, m.file_path, m.error, m.created_at, m.updated_at
		FROM meetings m
		LEFT JOIN transcripts t ON t.meeting_id = m.id
		LEFT JOIN summaries  s ON s.meeting_id = m.id
		WHERE m.user_id = $1
		  AND (
		      COALESCE(t.text, '') ILIKE '%' || $2 || '%'
		   OR COALESCE(s.text, '') ILIKE '%' || $2 || '%'
		  )
		ORDER BY m.created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Meeting
	for rows.Next() {
		var m domain.Meeting
		if err := rows.Scan(&m.ID, &m.UserID, &m.Title, &m.Status, &m.FilePath, &m.Error, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}
