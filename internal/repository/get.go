package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`

	user, err := scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("user with email %q: %w", email, models.ErrNotFound)
		}

		return models.User{}, fmt.Errorf("select user by email: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	const query = `SELECT id, email, password_hash, created_at FROM users WHERE id = $1`

	user, err := scanUser(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("user %s: %w", id, models.ErrNotFound)
		}

		return models.User{}, fmt.Errorf("select user by id: %w", err)
	}

	return user, nil
}

func (r *Repository) GetTask(ctx context.Context, id uuid.UUID) (models.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1`

	task, err := scanTask(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Task{}, fmt.Errorf("task %s: %w", id, models.ErrNotFound)
		}

		return models.Task{}, fmt.Errorf("select task: %w", err)
	}

	return task, nil
}
