package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `SELECT ` + columns + ` FROM users WHERE email = $1`

	user, err := scan(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("user with email %q: %w", email, models.ErrNotFound)
		}

		return models.User{}, fmt.Errorf("select user by email: %w", err)
	}

	return user, nil
}
