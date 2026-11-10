package tasks

import (
	"context"
	"fmt"
)

// MarkOverdue переводит в статус overdue невыполненные задачи с истёкшим сроком
// и возвращает число изменённых задач.
func (r *Repository) MarkOverdue(ctx context.Context) (int64, error) {
	const query = `
		UPDATE tasks
		SET status = 'overdue', updated_at = now()
		WHERE deadline < now() AND status IN ('created', 'in_progress')`

	tag, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("mark overdue tasks: %w", err)
	}

	return tag.RowsAffected(), nil
}
