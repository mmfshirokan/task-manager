package handlers

//go:generate go tool mockgen -source=handlers.go -destination=mocks/handlers_mock.go -package=mocks

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/internal/handlers/endpoints/auth"
	"github.com/mmfshirokan/task-manager/internal/handlers/endpoints/tasks"
	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (models.User, error)
	Login(ctx context.Context, email, password string) (models.Token, error)
	ParseToken(token string) (uuid.UUID, error)
}

type TaskService interface {
	Create(ctx context.Context, creatorID uuid.UUID, in models.NewTask) (models.Task, error)
	Get(ctx context.Context, id uuid.UUID) (models.Task, error)
	List(ctx context.Context, filter models.TaskFilter) (models.TaskPage, error)
	Update(ctx context.Context, userID, taskID uuid.UUID, upd models.TaskUpdate) (models.Task, error)
	Delete(ctx context.Context, userID, taskID uuid.UUID) error
}

type Handler struct {
	authSvc AuthService
	auth    *auth.Handler
	tasks   *tasks.Handler
	resp    *response.Writer
	log     *slog.Logger
}

func New(authSvc AuthService, taskSvc TaskService, log *slog.Logger) *Handler {
	resp := response.NewWriter(log)

	return &Handler{
		authSvc: authSvc,
		auth:    auth.New(authSvc, resp),
		tasks:   tasks.New(taskSvc, resp),
		resp:    resp,
		log:     log,
	}
}
