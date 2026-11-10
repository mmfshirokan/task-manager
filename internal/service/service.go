package service

import (
	"log/slog"

	"github.com/mmfshirokan/task-manager/internal/service/auth"
	"github.com/mmfshirokan/task-manager/internal/service/tasks"
)

// UserRepository объединяет то, что нужно от таблицы users сервисам auth и tasks.
type UserRepository interface {
	auth.UserRepository
	tasks.UserRepository
}

type Config = auth.Config

type Service struct {
	Auth  *auth.Service
	Tasks *tasks.Service
}

func New(users UserRepository, taskRepo tasks.TaskRepository, cfg Config, log *slog.Logger) *Service {
	return &Service{
		Auth:  auth.New(users, cfg, log),
		Tasks: tasks.New(taskRepo, users, log),
	}
}
