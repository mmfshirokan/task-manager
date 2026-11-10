package tasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (models.Task, error) {
	const query = `SELECT ` + columns + ` FROM tasks WHERE id = $1`

	task, err := scan(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Task{}, fmt.Errorf("task %s: %w", id, models.ErrNotFound)
		}

		return models.Task{}, fmt.Errorf("select task: %w", err)
	}

	return task, nil
}
