package tasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) Update(ctx context.Context, id uuid.UUID, upd models.TaskUpdate) (models.Task, error) {
	const query = `
		UPDATE tasks SET
			title       = COALESCE($2, title),
			description = COALESCE($3, description),
			deadline    = COALESCE($4, deadline),
			status      = COALESCE($5::task_status, status),
			assignee_id = COALESCE($6, assignee_id),
			updated_at  = now()
		WHERE id = $1
		RETURNING ` + columns

	task, err := scan(r.db.QueryRow(ctx, query,
		id, upd.Title, upd.Description, upd.Deadline, statusArg(upd.Status), upd.AssigneeID,
	))

	switch {
	case err == nil:
		return task, nil
	case errors.Is(err, pgx.ErrNoRows):
		return models.Task{}, fmt.Errorf("task %s: %w", id, models.ErrNotFound)
	case isForeignKeyViolation(err):
		return models.Task{}, errAssigneeNotFound
	default:
		return models.Task{}, fmt.Errorf("update task: %w", err)
	}
}
