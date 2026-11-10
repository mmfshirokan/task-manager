package auth

import (
	"context"

	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

type Service interface {
	Register(ctx context.Context, email, password string) (models.User, error)
	Login(ctx context.Context, email, password string) (models.Token, error)
}

type Handler struct {
	svc  Service
	resp *response.Writer
}

func New(svc Service, resp *response.Writer) *Handler {
	return &Handler{svc: svc, resp: resp}
}
