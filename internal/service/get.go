package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (models.Task, error) {
	return s.tasks.GetTask(ctx, id)
}
