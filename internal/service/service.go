package service

//go:generate go tool mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

const (
	maxTitleLength       = 200
	maxDescriptionLength = 5000
)

type UserRepository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (models.User, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error)
}

type TaskRepository interface {
	CreateTask(ctx context.Context, task models.Task) (models.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (models.Task, error)
	ListTasks(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error)
	UpdateTask(ctx context.Context, id uuid.UUID, upd models.TaskUpdate) (models.Task, error)
	DeleteTask(ctx context.Context, id uuid.UUID) error
	MarkOverdue(ctx context.Context) (int64, error)
}

type Config struct {
	JWTSecret []byte
	TokenTTL  time.Duration
}

type Service struct {
	users UserRepository
	tasks TaskRepository
	cfg   Config
	log   *slog.Logger
}

func New(users UserRepository, tasks TaskRepository, cfg Config, log *slog.Logger) *Service {
	return &Service{users: users, tasks: tasks, cfg: cfg, log: log}
}

func validateTitle(title string) error {
	switch {
	case strings.TrimSpace(title) == "":
		return &models.ValidationError{Field: "title", Message: "must not be empty"}
	case utf8.RuneCountInString(title) > maxTitleLength:
		return &models.ValidationError{Field: "title", Message: fmt.Sprintf("must be at most %d characters", maxTitleLength)}
	}

	return nil
}

func validateDescription(description string) error {
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return &models.ValidationError{
			Field:   "description",
			Message: fmt.Sprintf("must be at most %d characters", maxDescriptionLength),
		}
	}

	return nil
}

func validateStatus(status models.Status) error {
	if !status.Valid() {
		return &models.ValidationError{
			Field:   "status",
			Message: "must be one of: created, in_progress, completed, overdue",
		}
	}

	return nil
}

// ensureUserExists reports a validation error when the assignee is unknown.
func (s *Service) ensureUserExists(ctx context.Context, id uuid.UUID) error {
	_, err := s.users.GetUserByID(ctx, id)
	if errors.Is(err, models.ErrNotFound) {
		return &models.ValidationError{Field: "assignee_id", Message: "user does not exist"}
	}

	if err != nil {
		return fmt.Errorf("check assignee: %w", err)
	}

	return nil
}
