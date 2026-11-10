package tasks

import (
	"context"
	"fmt"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) Create(ctx context.Context, task models.Task) (models.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, deadline, status, creator_id, assignee_id)
		VALUES ($1, $2, $3, $4::task_status, $5, $6)
		RETURNING ` + columns

	created, err := scan(r.db.QueryRow(ctx, query,
		task.Title, task.Description, task.Deadline, string(task.Status), task.CreatorID, task.AssigneeID,
	))
	if err != nil {
		if isForeignKeyViolation(err) {
			return models.Task{}, errAssigneeNotFound
		}

		return models.Task{}, fmt.Errorf("insert task: %w", err)
	}

	return created, nil
}
