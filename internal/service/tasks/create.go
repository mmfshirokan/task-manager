package tasks

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

// Create создаёт задачу от имени creatorID. Если исполнитель не указан, им
// становится автор.
func (s *Service) Create(ctx context.Context, creatorID uuid.UUID, in models.NewTask) (models.Task, error) {
	if err := validateNewTask(in); err != nil {
		return models.Task{}, err
	}

	assigneeID := creatorID

	if in.AssigneeID != nil && *in.AssigneeID != creatorID {
		if err := s.ensureUserExists(ctx, *in.AssigneeID); err != nil {
			return models.Task{}, err
		}

		assigneeID = *in.AssigneeID
	}

	task, err := s.tasks.Create(ctx, models.Task{
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Deadline:    in.Deadline,
		Status:      models.StatusCreated,
		CreatorID:   creatorID,
		AssigneeID:  assigneeID,
	})
	if err != nil {
		return models.Task{}, err
	}

	s.log.InfoContext(ctx, "task created",
		"task_id", task.ID, "creator_id", task.CreatorID, "assignee_id", task.AssigneeID)

	return task, nil
}

func validateNewTask(in models.NewTask) error {
	if err := validateTitle(in.Title); err != nil {
		return err
	}

	if err := validateDescription(in.Description); err != nil {
		return err
	}

	switch {
	case in.Deadline.IsZero():
		return &models.ValidationError{Field: "deadline", Message: "is required"}
	case !in.Deadline.After(time.Now()):
		return &models.ValidationError{Field: "deadline", Message: "must be in the future"}
	}

	return nil
}
