package tasks

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

type Service interface {
	Create(ctx context.Context, creatorID uuid.UUID, in models.NewTask) (models.Task, error)
	Get(ctx context.Context, id uuid.UUID) (models.Task, error)
	List(ctx context.Context, filter models.TaskFilter) (models.TaskPage, error)
	Update(ctx context.Context, userID, taskID uuid.UUID, upd models.TaskUpdate) (models.Task, error)
	Delete(ctx context.Context, userID, taskID uuid.UUID) error
}

// Handler ожидает идентификатор пользователя в контексте запроса
// (его кладёт middleware.Authenticate).
type Handler struct {
	svc  Service
	resp *response.Writer
}

func New(svc Service, resp *response.Writer) *Handler {
	return &Handler{svc: svc, resp: resp}
}

func taskIDParam(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, &models.ValidationError{Field: "id", Message: "must be a valid UUID"}
	}

	return id, nil
}
