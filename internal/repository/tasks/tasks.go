package tasks

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mmfshirokan/task-manager/models"
)

const pgForeignKeyViolation = "23503"

const columns = `id, title, description, deadline, status::text, creator_id, assignee_id, created_at, updated_at`

var errAssigneeNotFound = &models.ValidationError{Field: "assignee_id", Message: "user does not exist"}

type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db DB
}

func New(db DB) *Repository {
	return &Repository{db: db}
}

func scan(row pgx.Row) (models.Task, error) {
	var (
		t      models.Task
		status string
	)

	err := row.Scan(
		&t.ID, &t.Title, &t.Description, &t.Deadline, &status,
		&t.CreatorID, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return models.Task{}, err
	}

	t.Status = models.Status(status)

	return t, nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation
}

func statusArg(s *models.Status) *string {
	if s == nil {
		return nil
	}

	v := string(*s)

	return &v
}
