package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusCreated    Status = "created"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusOverdue    Status = "overdue"
)

func (s Status) Valid() bool {
	switch s {
	case StatusCreated, StatusInProgress, StatusCompleted, StatusOverdue:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          uuid.UUID
	Title       string
	Description string
	Deadline    time.Time
	Status      Status
	CreatorID   uuid.UUID
	AssigneeID  uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask — данные для создания задачи. Если AssigneeID равен nil, исполнителем
// становится автор.
type NewTask struct {
	Title       string
	Description string
	Deadline    time.Time
	AssigneeID  *uuid.UUID
}

// TaskUpdate — частичное обновление: поля со значением nil не изменяются.
type TaskUpdate struct {
	Title       *string
	Description *string
	Deadline    *time.Time
	Status      *Status
	AssigneeID  *uuid.UUID
}

// Fields возвращает JSON-имена полей, заданных в обновлении.
func (u TaskUpdate) Fields() []string {
	var fields []string

	if u.Title != nil {
		fields = append(fields, "title")
	}

	if u.Description != nil {
		fields = append(fields, "description")
	}

	if u.Deadline != nil {
		fields = append(fields, "deadline")
	}

	if u.Status != nil {
		fields = append(fields, "status")
	}

	if u.AssigneeID != nil {
		fields = append(fields, "assignee_id")
	}

	return fields
}

type TaskFilter struct {
	Status       *Status
	AssigneeID   *uuid.UUID
	DeadlineFrom *time.Time
	DeadlineTo   *time.Time
	Limit        int
	Offset       int
}

type TaskPage struct {
	Items  []Task
	Total  int
	Limit  int
	Offset int
}
