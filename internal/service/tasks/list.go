package tasks

import (
	"context"
	"fmt"

	"github.com/mmfshirokan/task-manager/models"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

func (s *Service) List(ctx context.Context, filter models.TaskFilter) (models.TaskPage, error) {
	if filter.Limit == 0 {
		filter.Limit = defaultPageLimit
	}

	if err := validateFilter(filter); err != nil {
		return models.TaskPage{}, err
	}

	tasks, total, err := s.tasks.List(ctx, filter)
	if err != nil {
		return models.TaskPage{}, err
	}

	return models.TaskPage{Items: tasks, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func validateFilter(filter models.TaskFilter) error {
	if filter.Status != nil {
		if err := validateStatus(*filter.Status); err != nil {
			return err
		}
	}

	switch {
	case filter.Limit < 1 || filter.Limit > maxPageLimit:
		return &models.ValidationError{Field: "limit", Message: fmt.Sprintf("must be between 1 and %d", maxPageLimit)}
	case filter.Offset < 0:
		return &models.ValidationError{Field: "offset", Message: "must not be negative"}
	case filter.DeadlineFrom != nil && filter.DeadlineTo != nil && filter.DeadlineFrom.After(*filter.DeadlineTo):
		return &models.ValidationError{Field: "deadline_from", Message: "must not be after deadline_to"}
	}

	return nil
}
