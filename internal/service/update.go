package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

// UpdateTask applies a partial update on behalf of userID. The creator may
// change any field; the assignee may change only the status.
func (s *Service) UpdateTask(
	ctx context.Context, userID, taskID uuid.UUID, upd models.TaskUpdate,
) (models.Task, error) {
	if err := validateUpdate(upd); err != nil {
		return models.Task{}, err
	}

	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return models.Task{}, err
	}

	if err := authorizeUpdate(userID, task, upd); err != nil {
		s.log.WarnContext(ctx, "task update denied", "task_id", taskID, "user_id", userID, "error", err)

		return models.Task{}, err
	}

	if upd.AssigneeID != nil {
		if err := s.ensureUserExists(ctx, *upd.AssigneeID); err != nil {
			return models.Task{}, err
		}
	}

	if upd.Title != nil {
		title := strings.TrimSpace(*upd.Title)
		upd.Title = &title
	}

	updated, err := s.tasks.UpdateTask(ctx, taskID, upd)
	if err != nil {
		return models.Task{}, err
	}

	s.log.InfoContext(ctx, "task updated", "task_id", taskID, "user_id", userID, "fields", upd.Fields())

	return updated, nil
}

// MarkOverdue moves unfinished tasks with a passed deadline to the overdue
// status.
func (s *Service) MarkOverdue(ctx context.Context) (int64, error) {
	return s.tasks.MarkOverdue(ctx)
}

func validateUpdate(upd models.TaskUpdate) error {
	if len(upd.Fields()) == 0 {
		return &models.ValidationError{Field: "body", Message: "no fields to update"}
	}

	if upd.Title != nil {
		if err := validateTitle(*upd.Title); err != nil {
			return err
		}
	}

	if upd.Description != nil {
		if err := validateDescription(*upd.Description); err != nil {
			return err
		}
	}

	if upd.Status != nil {
		return validateStatus(*upd.Status)
	}

	return nil
}

func authorizeUpdate(userID uuid.UUID, task models.Task, upd models.TaskUpdate) error {
	if task.CreatorID == userID {
		return nil
	}

	if task.AssigneeID != userID {
		return fmt.Errorf("%w: only the creator or the assignee can update the task", models.ErrForbidden)
	}

	notAllowed := slices.DeleteFunc(upd.Fields(), func(f string) bool { return f == "status" })
	if len(notAllowed) > 0 {
		return fmt.Errorf("%w: the assignee can change only status, not: %s",
			models.ErrForbidden, strings.Join(notAllowed, ", "))
	}

	return nil
}
