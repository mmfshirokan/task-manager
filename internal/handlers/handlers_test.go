package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/mmfshirokan/task-manager/internal/handlers"
	"github.com/mmfshirokan/task-manager/internal/handlers/mocks"
	"github.com/mmfshirokan/task-manager/models"
)

const validToken = "valid-token"

var (
	currentUser = uuid.New()
	errDB       = errors.New("connection refused")
)

type deps struct {
	auth  *mocks.MockAuthService
	tasks *mocks.MockTaskService
}

type testCase struct {
	name       string
	method     string
	path       string
	body       string
	token      string
	setup      func(t *testing.T, d deps)
	wantStatus int
	wantBody   string
}

// run прогоняет каждый случай через весь роутер с моками сервисов. Запрос с
// validToken считается запросом пользователя currentUser.
func run(t *testing.T, tests []testCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			d := deps{auth: mocks.NewMockAuthService(ctrl), tasks: mocks.NewMockTaskService(ctrl)}
			d.auth.EXPECT().ParseToken(validToken).Return(currentUser, nil).AnyTimes()

			if tt.setup != nil {
				tt.setup(t, d)
			}

			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			rec := httptest.NewRecorder()
			handlers.New(d.auth, d.tasks, slog.New(slog.DiscardHandler)).Routes().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body)
			}

			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rec.Body, tt.wantBody)
			}
		})
	}
}

func sampleTask() models.Task {
	return models.Task{
		ID:         uuid.MustParse("6f1c3c9e-2f41-4c1a-9d0e-3a4c8a1b2c3d"),
		Title:      "Write docs",
		Deadline:   time.Date(2030, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:     models.StatusCreated,
		CreatorID:  currentUser,
		AssigneeID: currentUser,
	}
}

func TestHealthAndUnknownRoutes(t *testing.T) {
	run(t, []testCase{
		{name: "healthz", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK, wantBody: `"status":"ok"`},
		{name: "unknown route", method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound, wantBody: "route not found"},
		{
			name: "wrong method", method: http.MethodPut, path: "/api/v1/auth/login",
			wantStatus: http.StatusMethodNotAllowed, wantBody: "method not allowed",
		},
	})
}

func TestRegister(t *testing.T) {
	const path = "/api/v1/auth/register"

	body := `{"email":"alice@example.com","password":"secret123"}`

	run(t, []testCase{
		{
			name: "created", method: http.MethodPost, path: path, body: body,
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().Register(gomock.Any(), "alice@example.com", "secret123").
					Return(models.User{ID: currentUser, Email: "alice@example.com", PasswordHash: "hash"}, nil)
			},
			wantStatus: http.StatusCreated, wantBody: `"email":"alice@example.com"`,
		},
		{
			name: "malformed JSON", method: http.MethodPost, path: path, body: `{"email":`,
			wantStatus: http.StatusBadRequest, wantBody: "invalid JSON",
		},
		{
			name: "unknown field", method: http.MethodPost, path: path, body: `{"login":"alice"}`,
			wantStatus: http.StatusBadRequest, wantBody: "unknown field",
		},
		{
			name: "validation error", method: http.MethodPost, path: path, body: body,
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.User{}, &models.ValidationError{Field: "password", Message: "too short"})
			},
			wantStatus: http.StatusBadRequest, wantBody: "password: too short",
		},
		{
			name: "duplicate email", method: http.MethodPost, path: path, body: body,
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(models.User{}, fmt.Errorf("user with email %q: %w", "alice@example.com", models.ErrAlreadyExists))
			},
			wantStatus: http.StatusConflict, wantBody: "already exists",
		},
		{
			name: "internal error is hidden", method: http.MethodPost, path: path, body: body,
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.User{}, errDB)
			},
			wantStatus: http.StatusInternalServerError, wantBody: `{"error":"internal server error"}`,
		},
	})
}

func TestLogin(t *testing.T) {
	const path = "/api/v1/auth/login"

	body := `{"email":"alice@example.com","password":"secret123"}`

	run(t, []testCase{
		{
			name: "issues bearer token", method: http.MethodPost, path: path, body: body,
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().Login(gomock.Any(), "alice@example.com", "secret123").
					Return(models.Token{AccessToken: "jwt", ExpiresIn: time.Hour}, nil)
			},
			wantStatus: http.StatusOK, wantBody: `{"access_token":"jwt","token_type":"Bearer","expires_in":3600}`,
		},
		{
			name: "invalid credentials", method: http.MethodPost, path: path, body: body,
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Token{}, models.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized, wantBody: "invalid email or password",
		},
	})
}

func TestAuthentication(t *testing.T) {
	const path = "/api/v1/tasks/6f1c3c9e-2f41-4c1a-9d0e-3a4c8a1b2c3d"

	tests := []testCase{
		{name: "missing header", method: http.MethodGet, path: path, wantStatus: http.StatusUnauthorized, wantBody: "missing or malformed"},
		{
			name: "rejected token", method: http.MethodGet, path: path, token: "expired",
			setup: func(_ *testing.T, d deps) {
				d.auth.EXPECT().ParseToken("expired").Return(uuid.Nil, fmt.Errorf("%w: invalid or expired token", models.ErrUnauthorized))
			},
			wantStatus: http.StatusUnauthorized, wantBody: "invalid or expired token",
		},
	}
	run(t, tests)

	t.Run("non-bearer scheme", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		h := handlers.New(mocks.NewMockAuthService(ctrl), mocks.NewMockTaskService(ctrl), slog.New(slog.DiscardHandler))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Basic YWxpY2U6c2VjcmV0")

		rec := httptest.NewRecorder()
		h.Routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestCreateTask(t *testing.T) {
	const path = "/api/v1/tasks"

	assignee := uuid.New()
	body := fmt.Sprintf(`{"title":"Write docs","description":"API","deadline":"2030-01-02T15:04:05Z","assignee_id":%q}`, assignee)

	run(t, []testCase{
		{
			name: "created", method: http.MethodPost, path: path, body: body, token: validToken,
			setup: func(t *testing.T, d deps) {
				d.tasks.EXPECT().Create(gomock.Any(), currentUser, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ uuid.UUID, in models.NewTask) (models.Task, error) {
						wantDeadline := time.Date(2030, 1, 2, 15, 4, 5, 0, time.UTC)
						if in.Title != "Write docs" || in.Description != "API" || !in.Deadline.Equal(wantDeadline) ||
							in.AssigneeID == nil || *in.AssigneeID != assignee {
							t.Errorf("unexpected input: %+v", in)
						}

						return sampleTask(), nil
					})
			},
			wantStatus: http.StatusCreated, wantBody: `"status":"created"`,
		},
		{
			name: "validation error", method: http.MethodPost, path: path, body: `{"title":""}`, token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().Create(gomock.Any(), currentUser, gomock.Any()).
					Return(models.Task{}, &models.ValidationError{Field: "title", Message: "must not be empty"})
			},
			wantStatus: http.StatusBadRequest, wantBody: "title: must not be empty",
		},
		{
			name: "invalid assignee id", method: http.MethodPost, path: path, body: `{"assignee_id":"x"}`, token: validToken,
			wantStatus: http.StatusBadRequest, wantBody: "invalid JSON",
		},
	})
}

func TestGetTask(t *testing.T) {
	task := sampleTask()
	path := "/api/v1/tasks/" + task.ID.String()

	run(t, []testCase{
		{
			name: "found", method: http.MethodGet, path: path, token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().Get(gomock.Any(), task.ID).Return(task, nil)
			},
			wantStatus: http.StatusOK, wantBody: `"title":"Write docs"`,
		},
		{
			name: "not found", method: http.MethodGet, path: path, token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().Get(gomock.Any(), task.ID).Return(models.Task{}, fmt.Errorf("task: %w", models.ErrNotFound))
			},
			wantStatus: http.StatusNotFound, wantBody: "not found",
		},
		{
			name: "invalid id", method: http.MethodGet, path: "/api/v1/tasks/42", token: validToken,
			wantStatus: http.StatusBadRequest, wantBody: "id: must be a valid UUID",
		},
	})
}

func TestListTasks(t *testing.T) {
	assignee, creator := uuid.New(), uuid.New()
	query := fmt.Sprintf("/api/v1/tasks?status=in_progress&assignee_id=%s&creator_id=%s"+
		"&deadline_from=2030-01-01T00:00:00Z&deadline_to=2030-02-01T00:00:00Z&limit=5&offset=10", assignee, creator)

	run(t, []testCase{
		{
			name: "parses filters", method: http.MethodGet, path: query, token: validToken,
			setup: func(t *testing.T, d deps) {
				d.tasks.EXPECT().List(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, f models.TaskFilter) (models.TaskPage, error) {
						checkFilter(t, f, assignee, creator)

						return models.TaskPage{Items: []models.Task{sampleTask()}, Total: 11, Limit: 5, Offset: 10}, nil
					})
			},
			wantStatus: http.StatusOK, wantBody: `"total":11,"limit":5,"offset":10`,
		},
		{
			name: "empty page is an empty array", method: http.MethodGet, path: "/api/v1/tasks", token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().List(gomock.Any(), models.TaskFilter{}).Return(models.TaskPage{Limit: 20}, nil)
			},
			wantStatus: http.StatusOK, wantBody: `"items":[]`,
		},
		{
			name: "invalid limit", method: http.MethodGet, path: "/api/v1/tasks?limit=ten", token: validToken,
			wantStatus: http.StatusBadRequest, wantBody: "limit: must be an integer",
		},
		{
			name: "invalid assignee", method: http.MethodGet, path: "/api/v1/tasks?assignee_id=bob", token: validToken,
			wantStatus: http.StatusBadRequest, wantBody: "assignee_id: must be a valid UUID",
		},
		{
			name: "invalid creator", method: http.MethodGet, path: "/api/v1/tasks?creator_id=alice", token: validToken,
			wantStatus: http.StatusBadRequest, wantBody: "creator_id: must be a valid UUID",
		},
		{
			name: "invalid deadline", method: http.MethodGet, path: "/api/v1/tasks?deadline_to=tomorrow", token: validToken,
			wantStatus: http.StatusBadRequest, wantBody: "deadline_to: must be an RFC 3339 timestamp",
		},
	})
}

func checkFilter(t *testing.T, f models.TaskFilter, assignee, creator uuid.UUID) {
	t.Helper()

	wantFrom := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)

	switch {
	case f.Status == nil || *f.Status != models.StatusInProgress:
		t.Errorf("status = %v, want in_progress", f.Status)
	case f.AssigneeID == nil || *f.AssigneeID != assignee:
		t.Errorf("assignee = %v, want %s", f.AssigneeID, assignee)
	case f.CreatorID == nil || *f.CreatorID != creator:
		t.Errorf("creator = %v, want %s", f.CreatorID, creator)
	case f.DeadlineFrom == nil || !f.DeadlineFrom.Equal(wantFrom):
		t.Errorf("deadline_from = %v, want %s", f.DeadlineFrom, wantFrom)
	case f.DeadlineTo == nil || !f.DeadlineTo.Equal(wantTo):
		t.Errorf("deadline_to = %v, want %s", f.DeadlineTo, wantTo)
	case f.Limit != 5 || f.Offset != 10:
		t.Errorf("limit/offset = %d/%d, want 5/10", f.Limit, f.Offset)
	}
}

func TestUpdateTask(t *testing.T) {
	task := sampleTask()
	path := "/api/v1/tasks/" + task.ID.String()

	run(t, []testCase{
		{
			name: "updated", method: http.MethodPatch, path: path, body: `{"status":"completed"}`, token: validToken,
			setup: func(t *testing.T, d deps) {
				d.tasks.EXPECT().Update(gomock.Any(), currentUser, task.ID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ uuid.UUID, upd models.TaskUpdate) (models.Task, error) {
						if upd.Status == nil || *upd.Status != models.StatusCompleted || len(upd.Fields()) != 1 {
							t.Errorf("unexpected update: %v", upd.Fields())
						}

						updated := task
						updated.Status = models.StatusCompleted

						return updated, nil
					})
			},
			wantStatus: http.StatusOK, wantBody: `"status":"completed"`,
		},
		{
			name: "forbidden", method: http.MethodPatch, path: path, body: `{"title":"x"}`, token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().Update(gomock.Any(), currentUser, task.ID, gomock.Any()).
					Return(models.Task{}, fmt.Errorf("%w: the assignee can change only status, not: title", models.ErrForbidden))
			},
			wantStatus: http.StatusForbidden, wantBody: "the assignee can change only status",
		},
	})
}

func TestDeleteTask(t *testing.T) {
	task := sampleTask()
	path := "/api/v1/tasks/" + task.ID.String()

	run(t, []testCase{
		{
			name: "deleted", method: http.MethodDelete, path: path, token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().Delete(gomock.Any(), currentUser, task.ID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "forbidden", method: http.MethodDelete, path: path, token: validToken,
			setup: func(_ *testing.T, d deps) {
				d.tasks.EXPECT().Delete(gomock.Any(), currentUser, task.ID).
					Return(fmt.Errorf("%w: only the creator can delete the task", models.ErrForbidden))
			},
			wantStatus: http.StatusForbidden, wantBody: "only the creator can delete the task",
		},
	})
}

func TestTaskResponseShape(t *testing.T) {
	ctrl := gomock.NewController(t)
	auth, tasks := mocks.NewMockAuthService(ctrl), mocks.NewMockTaskService(ctrl)
	task := sampleTask()

	auth.EXPECT().ParseToken(validToken).Return(currentUser, nil)
	tasks.EXPECT().Get(gomock.Any(), task.ID).Return(task, nil)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/tasks/"+task.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	rec := httptest.NewRecorder()
	handlers.New(auth, tasks, slog.New(slog.DiscardHandler)).Routes().ServeHTTP(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"id", "title", "description", "deadline", "status", "creator_id", "assignee_id", "created_at", "updated_at"} {
		if _, ok := got[key]; !ok {
			t.Errorf("response has no %q field: %v", key, got)
		}
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
