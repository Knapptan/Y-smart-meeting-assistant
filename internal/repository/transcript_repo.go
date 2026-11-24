package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// TranscriptRepo — реализация domain.TranscriptRepository.
type TranscriptRepo struct {
	db *DB
}

// NewTranscriptRepo создаёт репозиторий транскрипций.
func NewTranscriptRepo(db *DB) *TranscriptRepo {
	return &TranscriptRepo{db: db}
}

// Save атомарно сохраняет транскрипцию и переводит встречу в статус transcribed.
func (r *TranscriptRepo) Save(ctx context.Context, t *domain.Transcript) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO transcripts (meeting_id, text) VALUES ($1, $2)
		 ON CONFLICT (meeting_id) DO UPDATE SET text = EXCLUDED.text`,
		t.MeetingID, t.Text)
	if err != nil {
		return fmt.Errorf("insert transcript: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE meetings SET status = 'transcribed', updated_at = NOW() WHERE id = $1`,
		t.MeetingID)
	if err != nil {
		return fmt.Errorf("update meeting status: %w", err)
	}
	return tx.Commit()
}

// GetByMeetingID возвращает транскрипцию.
func (r *TranscriptRepo) GetByMeetingID(ctx context.Context, meetingID int64) (*domain.Transcript, error) {
	const q = `SELECT meeting_id, text, created_at FROM transcripts WHERE meeting_id = $1`
	var t domain.Transcript
	err := r.db.QueryRowContext(ctx, q, meetingID).Scan(&t.MeetingID, &t.Text, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}
