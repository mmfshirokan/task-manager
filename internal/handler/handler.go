package handler

//go:generate go tool mockgen -source=handler.go -destination=mocks/handler_mock.go -package=mocks

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (models.User, error)
	Login(ctx context.Context, email, password string) (models.Token, error)
	ParseToken(token string) (uuid.UUID, error)
}

type TaskService interface {
	CreateTask(ctx context.Context, creatorID uuid.UUID, in models.NewTask) (models.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (models.Task, error)
	ListTasks(ctx context.Context, filter models.TaskFilter) (models.TaskPage, error)
	UpdateTask(ctx context.Context, userID, taskID uuid.UUID, upd models.TaskUpdate) (models.Task, error)
	DeleteTask(ctx context.Context, userID, taskID uuid.UUID) error
}

type Handler struct {
	auth  AuthService
	tasks TaskService
	log   *slog.Logger
}

func New(auth AuthService, tasks TaskService, log *slog.Logger) *Handler {
	return &Handler{auth: auth, tasks: tasks, log: log}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID, h.logRequests, chimiddleware.Recoverer)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.writeJSON(w, r, http.StatusNotFound, errorResponse{Error: "route not found"})
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		h.writeJSON(w, r, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
	})

	r.Get("/healthz", h.healthz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", h.register)
		r.Post("/auth/login", h.login)

		r.Group(func(r chi.Router) {
			r.Use(h.authenticate)

			r.Post("/tasks", h.createTask)
			r.Get("/tasks", h.listTasks)
			r.Get("/tasks/{id}", h.getTask)
			r.Patch("/tasks/{id}", h.updateTask)
			r.Delete("/tasks/{id}", h.deleteTask)
		})
	})

	return r
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}
