package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// MeetingRepo — реализация domain.MeetingRepository.
type MeetingRepo struct {
	db *DB
}

// NewMeetingRepo создаёт репозиторий встреч.
func NewMeetingRepo(db *DB) *MeetingRepo {
	return &MeetingRepo{db: db}
}

// CreateMeetingWithTask атомарно создаёт встречу и задачу обработки.
// Используется транзакция: если задача не создастся, встреча тоже откатится.
func (r *MeetingRepo) CreateMeetingWithTask(ctx context.Context, m *domain.Meeting) (*domain.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const insertMeeting = `
		INSERT INTO meetings (user_id, title, status, file_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	err = tx.QueryRowContext(ctx, insertMeeting, m.UserID, m.Title, m.Status, m.FilePath).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert meeting: %w", err)
	}

	const insertTask = `
		INSERT INTO tasks (meeting_id, status, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	t := &domain.Task{MeetingID: m.ID, Status: domain.StatusCreated}
	err = tx.QueryRowContext(ctx, insertTask, m.ID, t.Status).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return t, nil
}

// GetByID возвращает встречу без проверки пользователя.
func (r *MeetingRepo) GetByID(ctx context.Context, id int64) (*domain.Meeting, error) {
	const q = `
		SELECT id, user_id, title, status, file_path, error, created_at, updated_at
		FROM meetings WHERE id = $1`
	var m domain.Meeting
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&m.ID, &m.UserID, &m.Title, &m.Status, &m.FilePath, &m.Error, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByIDForUser возвращает встречу, только если она принадлежит userID.
func (r *MeetingRepo) GetByIDForUser(ctx context.Context, id, userID int64) (*domain.Meeting, error) {
	const q = `
		SELECT id, user_id, title, status, file_path, error, created_at, updated_at
		FROM meetings WHERE id = $1 AND user_id = $2`
	var m domain.Meeting
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(
		&m.ID, &m.UserID, &m.Title, &m.Status, &m.FilePath, &m.Error, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListByUser возвращает встречи пользователя, отсортированные по убыванию даты.
func (r *MeetingRepo) ListByUser(ctx context.Context, userID int64) ([]*domain.Meeting, error) {
	const q = `
		SELECT id, user_id, title, status, file_path, error, created_at, updated_at
		FROM meetings WHERE user_id = $1
		ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
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

// UpdateStatus обновляет статус встречи и updated_at.
func (r *MeetingRepo) UpdateStatus(ctx context.Context, id int64, s domain.Status, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE meetings SET status = $1, error = $2, updated_at = $3 WHERE id = $4`,
		s, errMsg, time.Now(), id)
	return err
}
