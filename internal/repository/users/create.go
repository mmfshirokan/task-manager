package users

import (
	"context"
	"fmt"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) Create(ctx context.Context, email, passwordHash string) (models.User, error) {
	const query = `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING ` + columns

	user, err := scan(r.db.QueryRow(ctx, query, email, passwordHash))
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, fmt.Errorf("user with email %q: %w", email, models.ErrAlreadyExists)
		}

		return models.User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}
