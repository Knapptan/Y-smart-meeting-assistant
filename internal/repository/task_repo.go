package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// TaskRepo — реализация domain.TaskRepository.
type TaskRepo struct {
	db *DB
}

// NewTaskRepo создаёт репозиторий задач.
func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// NextPending атомарно забирает одну задачу из очереди.
// Благодаря FOR UPDATE SKIP LOCKED несколько воркеров могут работать
// параллельно, не блокируя друг друга.
func (r *TaskRepo) NextPending(ctx context.Context) (*domain.Task, *domain.Meeting, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	const qSelect = `
		SELECT id, meeting_id, status, attempts
		FROM tasks
		WHERE status = 'created'
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`
	var t domain.Task
	err = tx.QueryRowContext(ctx, qSelect).Scan(&t.ID, &t.MeetingID, &t.Status, &t.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil // очередь пуста
	}
	if err != nil {
		return nil, nil, err
	}

	// Переводим задачу в processing и встречу в processing.
	const qUpdateTask = `UPDATE tasks SET status = 'processing', updated_at = $1 WHERE id = $2`
	if _, err := tx.ExecContext(ctx, qUpdateTask, time.Now(), t.ID); err != nil {
		return nil, nil, err
	}
	const qUpdateMeeting = `UPDATE meetings SET status = 'processing', updated_at = $1 WHERE id = $2`
	if _, err := tx.ExecContext(ctx, qUpdateMeeting, time.Now(), t.MeetingID); err != nil {
		return nil, nil, err
	}

	// Забираем встречу.
	const qMeeting = `
		SELECT id, user_id, title, status, file_path, error, created_at, updated_at
		FROM meetings WHERE id = $1`
	var m domain.Meeting
	err = tx.QueryRowContext(ctx, qMeeting, t.MeetingID).Scan(
		&m.ID, &m.UserID, &m.Title, &m.Status, &m.FilePath, &m.Error, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	t.Status = domain.StatusProcessing
	m.Status = domain.StatusProcessing
	return &t, &m, nil
}

// UpdateStatus обновляет статус задачи.
func (r *TaskRepo) UpdateStatus(ctx context.Context, id int64, s domain.Status, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = $1, error = $2, updated_at = $3 WHERE id = $4`,
		s, errMsg, time.Now(), id)
	return err
}

// IncAttempts увеличивает счётчик попыток.
func (r *TaskRepo) IncAttempts(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET attempts = attempts + 1, updated_at = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

// Retry атомарно переводит задачу встречи из failed в created
// и инкрементирует attempts. Возвращает nil, если задача не в failed
// или attempts >= maxAttempts.
func (r *TaskRepo) Retry(ctx context.Context, meetingID int64, maxAttempts int) (*domain.Task, error) {
	const q = `
		UPDATE tasks
		SET status = 'created', error = '', attempts = attempts + 1, updated_at = NOW()
		WHERE meeting_id = $1
		  AND status = 'failed'
		  AND attempts < $2
		RETURNING id, meeting_id, status, attempts, error, created_at, updated_at`
	var t domain.Task
	err := r.db.QueryRowContext(ctx, q, meetingID, maxAttempts).Scan(
		&t.ID, &t.MeetingID, &t.Status, &t.Attempts, &t.Error, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
