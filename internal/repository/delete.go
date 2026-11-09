package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) DeleteTask(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("task %s: %w", id, models.ErrNotFound)
	}

	return nil
}
