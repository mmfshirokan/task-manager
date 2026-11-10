package tasks

import (
	"context"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

func (s *Service) Get(ctx context.Context, id uuid.UUID) (models.Task, error) {
	return s.tasks.Get(ctx, id)
}
