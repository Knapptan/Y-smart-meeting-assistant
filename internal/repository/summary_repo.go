package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// SummaryRepo — реализация domain.SummaryRepository.
type SummaryRepo struct {
	db *DB
}

// NewSummaryRepo создаёт репозиторий выжимок.
func NewSummaryRepo(db *DB) *SummaryRepo {
	return &SummaryRepo{db: db}
}

// Save атомарно сохраняет выжимку и переводит встречу в статус summarized.
func (r *SummaryRepo) Save(ctx context.Context, s *domain.Summary) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO summaries (meeting_id, text) VALUES ($1, $2)
		 ON CONFLICT (meeting_id) DO UPDATE SET text = EXCLUDED.text`,
		s.MeetingID, s.Text)
	if err != nil {
		return fmt.Errorf("insert summary: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE meetings SET status = 'summarized', updated_at = NOW() WHERE id = $1`,
		s.MeetingID)
	if err != nil {
		return fmt.Errorf("update meeting status: %w", err)
	}
	return tx.Commit()
}

// GetByMeetingID возвращает выжимку.
func (r *SummaryRepo) GetByMeetingID(ctx context.Context, meetingID int64) (*domain.Summary, error) {
	const q = `SELECT meeting_id, text, created_at FROM summaries WHERE meeting_id = $1`
	var s domain.Summary
	err := r.db.QueryRowContext(ctx, q, meetingID).Scan(&s.MeetingID, &s.Text, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}
