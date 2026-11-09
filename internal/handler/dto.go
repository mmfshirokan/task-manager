package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

const maxBodyBytes = 1 << 20

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type createTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Deadline    time.Time  `json:"deadline"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
}

type updateTaskRequest struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Deadline    *time.Time     `json:"deadline"`
	Status      *models.Status `json:"status"`
	AssigneeID  *uuid.UUID     `json:"assignee_id"`
}

type taskResponse struct {
	ID          uuid.UUID     `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Deadline    time.Time     `json:"deadline"`
	Status      models.Status `json:"status"`
	CreatorID   uuid.UUID     `json:"creator_id"`
	AssigneeID  uuid.UUID     `json:"assignee_id"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type taskListResponse struct {
	Items  []taskResponse `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return &models.ValidationError{Field: "body", Message: "invalid JSON: " + err.Error()}
	}

	return nil
}

func toUserResponse(u models.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}

func toTaskResponse(t models.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Deadline:    t.Deadline,
		Status:      t.Status,
		CreatorID:   t.CreatorID,
		AssigneeID:  t.AssigneeID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func (req createTaskRequest) toModel() models.NewTask {
	return models.NewTask{
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
		AssigneeID:  req.AssigneeID,
	}
}

func (req updateTaskRequest) toModel() models.TaskUpdate {
	return models.TaskUpdate{
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	}
}
