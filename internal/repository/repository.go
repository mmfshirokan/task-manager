package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mmfshirokan/task-manager/models"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

const taskColumns = `id, title, description, deadline, status::text, creator_id, assignee_id, created_at, updated_at`

// DB is the subset of pgx used by the repository. It is satisfied by
// *pgxpool.Pool, *pgx.Conn and pgx.Tx.
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

func scanTask(row pgx.Row) (models.Task, error) {
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

func scanUser(row pgx.Row) (models.User, error) {
	var u models.User

	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	return u, err
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}

	return ""
}

func statusArg(s *models.Status) *string {
	if s == nil {
		return nil
	}

	v := string(*s)

	return &v
}

var errAssigneeNotFound = &models.ValidationError{Field: "assignee_id", Message: "user does not exist"}
