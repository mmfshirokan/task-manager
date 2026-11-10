package tasks

//go:generate go tool mockgen -source=tasks.go -destination=mocks/tasks_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

const (
	maxTitleLength       = 200
	maxDescriptionLength = 5000
)

type TaskRepository interface {
	Create(ctx context.Context, task models.Task) (models.Task, error)
	Get(ctx context.Context, id uuid.UUID) (models.Task, error)
	List(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error)
	Update(ctx context.Context, id uuid.UUID, upd models.TaskUpdate) (models.Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MarkOverdue(ctx context.Context) (int64, error)
}

// UserRepository нужен, чтобы проверить, что исполнитель существует.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
}

type Service struct {
	tasks TaskRepository
	users UserRepository
	log   *slog.Logger
}

func New(tasks TaskRepository, users UserRepository, log *slog.Logger) *Service {
	return &Service{tasks: tasks, users: users, log: log}
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

// ensureUserExists возвращает ошибку валидации, если исполнитель не найден.
func (s *Service) ensureUserExists(ctx context.Context, id uuid.UUID) error {
	_, err := s.users.GetByID(ctx, id)
	if errors.Is(err, models.ErrNotFound) {
		return &models.ValidationError{Field: "assignee_id", Message: "user does not exist"}
	}

	if err != nil {
		return fmt.Errorf("check assignee: %w", err)
	}

	return nil
}
