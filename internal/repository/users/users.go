package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mmfshirokan/task-manager/models"
)

const pgUniqueViolation = "23505"

const columns = `id, email, password_hash, created_at`

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

func scan(row pgx.Row) (models.User, error) {
	var u models.User

	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	return u, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
