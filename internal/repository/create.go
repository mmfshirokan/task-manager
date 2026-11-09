package repository

import (
	"context"
	"fmt"

	"github.com/mmfshirokan/task-manager/models"
)

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (models.User, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at`

	user, err := scanUser(r.db.QueryRow(ctx, query, email, passwordHash))
	if err != nil {
		if pgErrorCode(err) == pgUniqueViolation {
			return models.User{}, fmt.Errorf("user with email %q: %w", email, models.ErrAlreadyExists)
		}

		return models.User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (r *Repository) CreateTask(ctx context.Context, task models.Task) (models.Task, error) {
	query := `
		INSERT INTO tasks (title, description, deadline, status, creator_id, assignee_id)
		VALUES ($1, $2, $3, $4::task_status, $5, $6)
		RETURNING ` + taskColumns

	created, err := scanTask(r.db.QueryRow(ctx, query,
		task.Title, task.Description, task.Deadline, string(task.Status), task.CreatorID, task.AssigneeID,
	))
	if err != nil {
		if pgErrorCode(err) == pgForeignKeyViolation {
			return models.Task{}, errAssigneeNotFound
		}

		return models.Task{}, fmt.Errorf("insert task: %w", err)
	}

	return created, nil
}
