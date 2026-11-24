package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// UserRepo — реализация domain.UserRepository.
type UserRepo struct {
	db *DB
}

// NewUserRepo создаёт репозиторий пользователей.
func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

// GetOrCreate возвращает пользователя по external_id или создаёт нового.
func (r *UserRepo) GetOrCreate(ctx context.Context, externalID string) (*domain.User, error) {
	const q = `
		INSERT INTO users (external_id)
		VALUES ($1)
		ON CONFLICT (external_id) DO UPDATE SET external_id = EXCLUDED.external_id
		RETURNING id, external_id, created_at`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, externalID).Scan(&u.ID, &u.ExternalID, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get or create user: %w", err)
		}
		return nil, err
	}
	return &u, nil
}

// GetByExternalID ищет пользователя по external_id.
func (r *UserRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.User, error) {
	const q = `SELECT id, external_id, created_at FROM users WHERE external_id = $1`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, externalID).Scan(&u.ID, &u.ExternalID, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// touch обновляет updated_at у встречи (вспомогательная функция).
func touchMeeting(ctx context.Context, tx *sql.Tx, meetingID int64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE meetings SET updated_at = $1 WHERE id = $2`,
		time.Now(), meetingID)
	return err
}
