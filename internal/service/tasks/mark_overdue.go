package tasks

import (
	"context"
)

// MarkOverdue переводит невыполненные задачи с истёкшим сроком в статус overdue.
func (s *Service) MarkOverdue(ctx context.Context) (int64, error) {
	return s.tasks.MarkOverdue(ctx)
}
