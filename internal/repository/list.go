package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmfshirokan/task-manager/models"
)

// ListTasks returns one page of tasks matching the filter, newest first, and
// the total number of matching tasks.
func (r *Repository) ListTasks(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error) {
	where, args := buildTaskWhere(filter)

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM tasks`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	args = append(args, filter.Limit, filter.Offset)
	query := fmt.Sprintf(`SELECT %s FROM tasks%s ORDER BY created_at DESC, id LIMIT $%d OFFSET $%d`,
		taskColumns, where, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]models.Task, 0, filter.Limit)

	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, total, nil
}

func buildTaskWhere(filter models.TaskFilter) (string, []any) {
	var (
		conds []string
		args  []any
	)

	add := func(cond string, arg any) {
		args = append(args, arg)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if filter.Status != nil {
		add("status = $%d::task_status", string(*filter.Status))
	}

	if filter.AssigneeID != nil {
		add("assignee_id = $%d", *filter.AssigneeID)
	}

	if filter.DeadlineFrom != nil {
		add("deadline >= $%d", *filter.DeadlineFrom)
	}

	if filter.DeadlineTo != nil {
		add("deadline <= $%d", *filter.DeadlineTo)
	}

	if len(conds) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}
