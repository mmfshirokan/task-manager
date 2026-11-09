package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mmfshirokan/task-manager/models"
)

// UpdateTask applies the non-nil fields of upd to the task.
func (r *Repository) UpdateTask(ctx context.Context, id uuid.UUID, upd models.TaskUpdate) (models.Task, error) {
	query := `
		UPDATE tasks SET
			title       = COALESCE($2, title),
			description = COALESCE($3, description),
			deadline    = COALESCE($4, deadline),
			status      = COALESCE($5::task_status, status),
			assignee_id = COALESCE($6, assignee_id),
			updated_at  = now()
		WHERE id = $1
		RETURNING ` + taskColumns

	task, err := scanTask(r.db.QueryRow(ctx, query,
		id, upd.Title, upd.Description, upd.Deadline, statusArg(upd.Status), upd.AssigneeID,
	))

	switch {
	case err == nil:
		return task, nil
	case errors.Is(err, pgx.ErrNoRows):
		return models.Task{}, fmt.Errorf("task %s: %w", id, models.ErrNotFound)
	case pgErrorCode(err) == pgForeignKeyViolation:
		return models.Task{}, errAssigneeNotFound
	default:
		return models.Task{}, fmt.Errorf("update task: %w", err)
	}
}

// MarkOverdue sets the overdue status on unfinished tasks whose deadline has
// passed and returns how many tasks were changed.
func (r *Repository) MarkOverdue(ctx context.Context) (int64, error) {
	const query = `
		UPDATE tasks
		SET status = 'overdue', updated_at = now()
		WHERE deadline < now() AND status IN ('created', 'in_progress')`

	tag, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("mark overdue tasks: %w", err)
	}

	return tag.RowsAffected(), nil
}
