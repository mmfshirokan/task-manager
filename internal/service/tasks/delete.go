package tasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

// Delete удаляет задачу от имени userID. Удалить задачу может только автор.
func (s *Service) Delete(ctx context.Context, userID, taskID uuid.UUID) error {
	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}

	if task.CreatorID != userID {
		s.log.WarnContext(ctx, "task delete denied", "task_id", taskID, "user_id", userID)

		return fmt.Errorf("%w: only the creator can delete the task", models.ErrForbidden)
	}

	if err := s.tasks.Delete(ctx, taskID); err != nil {
		return err
	}

	s.log.InfoContext(ctx, "task deleted", "task_id", taskID, "user_id", userID)

	return nil
}
