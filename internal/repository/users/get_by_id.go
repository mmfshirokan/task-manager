package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	const query = `SELECT ` + columns + ` FROM users WHERE id = $1`

	user, err := scan(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("user %s: %w", id, models.ErrNotFound)
		}

		return models.User{}, fmt.Errorf("select user by id: %w", err)
	}

	return user, nil
}
