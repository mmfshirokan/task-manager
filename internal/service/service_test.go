package service_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/mmfshirokan/task-manager/internal/service"
	"github.com/mmfshirokan/task-manager/internal/service/mocks"
	"github.com/mmfshirokan/task-manager/models"
)

var (
	testSecret = []byte("test-secret")
	errDB      = errors.New("db is down")
)

type deps struct {
	users *mocks.MockUserRepository
	tasks *mocks.MockTaskRepository
}

func newService(t *testing.T) (*service.Service, deps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	d := deps{
		users: mocks.NewMockUserRepository(ctrl),
		tasks: mocks.NewMockTaskRepository(ctrl),
	}
	cfg := service.Config{JWTSecret: testSecret, TokenTTL: time.Hour}

	return service.New(d.users, d.tasks, cfg, slog.New(slog.DiscardHandler)), d
}

// checkErr asserts the error outcome: a validation error on wantField, an
// error matching wantErr, or no error at all.
func checkErr(t *testing.T, err, wantErr error, wantField string) {
	t.Helper()

	switch {
	case wantField != "":
		var validationErr *models.ValidationError
		if !errors.As(err, &validationErr) || validationErr.Field != wantField {
			t.Fatalf("want validation error on %q, got %v", wantField, err)
		}
	case wantErr != nil:
		if !errors.Is(err, wantErr) {
			t.Fatalf("want error %v, got %v", wantErr, err)
		}
	default:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		password  string
		setup     func(t *testing.T, d deps)
		wantField string
		wantErr   error
	}{
		{
			name:     "normalizes email and hashes password",
			email:    "  Alice@Example.COM ",
			password: "secret123",
			setup: func(t *testing.T, d deps) {
				d.users.EXPECT().CreateUser(gomock.Any(), "alice@example.com", gomock.Any()).
					DoAndReturn(func(_ context.Context, email, hash string) (models.User, error) {
						if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret123")); err != nil {
							t.Errorf("password is not stored as a bcrypt hash: %v", err)
						}

						return models.User{ID: uuid.New(), Email: email}, nil
					})
			},
		},
		{name: "invalid email", email: "not-an-email", password: "secret123", wantField: "email"},
		{name: "short password", email: "bob@example.com", password: "short", wantField: "password"},
		{name: "too long password", email: "bob@example.com", password: strings.Repeat("x", 73), wantField: "password"},
		{
			name:     "duplicate email",
			email:    "bob@example.com",
			password: "secret123",
			setup: func(_ *testing.T, d deps) {
				d.users.EXPECT().CreateUser(gomock.Any(), "bob@example.com", gomock.Any()).
					Return(models.User{}, fmt.Errorf("user: %w", models.ErrAlreadyExists))
			},
			wantErr: models.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newService(t)
			if tt.setup != nil {
				tt.setup(t, d)
			}

			_, err := svc.Register(t.Context(), tt.email, tt.password)
			checkErr(t, err, tt.wantErr, tt.wantField)
		})
	}
}

func TestLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	user := models.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: string(hash)}

	tests := []struct {
		name     string
		password string
		repoUser models.User
		repoErr  error
		wantErr  error
	}{
		{name: "valid credentials", password: "secret123", repoUser: user},
		{name: "wrong password", password: "wrong-password", repoUser: user, wantErr: models.ErrInvalidCredentials},
		{name: "unknown email", password: "secret123", repoErr: models.ErrNotFound, wantErr: models.ErrInvalidCredentials},
		{name: "repository failure", password: "secret123", repoErr: errDB, wantErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newService(t)
			d.users.EXPECT().GetUserByEmail(gomock.Any(), "alice@example.com").Return(tt.repoUser, tt.repoErr)

			token, err := svc.Login(t.Context(), " ALICE@example.com", tt.password)
			checkErr(t, err, tt.wantErr, "")

			if tt.wantErr != nil {
				return
			}

			if token.ExpiresIn != time.Hour {
				t.Errorf("ExpiresIn = %s, want 1h", token.ExpiresIn)
			}

			userID, err := svc.ParseToken(token.AccessToken)
			if err != nil {
				t.Fatalf("issued token is rejected: %v", err)
			}

			if userID != user.ID {
				t.Errorf("token subject = %s, want %s", userID, user.ID)
			}
		})
	}
}

func TestParseTokenRejectsInvalidTokens(t *testing.T) {
	sign := func(t *testing.T, method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
		t.Helper()

		token, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}

		return token
	}
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))
	past := jwt.NewNumericDate(time.Now().Add(-time.Hour))
	subject := uuid.NewString()

	tests := []struct {
		name  string
		token func(t *testing.T) string
	}{
		{name: "garbage", token: func(*testing.T) string { return "not-a-jwt" }},
		{name: "wrong secret", token: func(t *testing.T) string {
			return sign(t, jwt.SigningMethodHS256, []byte("other"), jwt.RegisteredClaims{Subject: subject, ExpiresAt: future})
		}},
		{name: "expired", token: func(t *testing.T) string {
			return sign(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: subject, ExpiresAt: past})
		}},
		{name: "without expiration", token: func(t *testing.T) string {
			return sign(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: subject})
		}},
		{name: "unexpected algorithm", token: func(t *testing.T) string {
			return sign(t, jwt.SigningMethodHS512, testSecret, jwt.RegisteredClaims{Subject: subject, ExpiresAt: future})
		}},
		{name: "subject is not a user id", token: func(t *testing.T) string {
			return sign(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: "admin", ExpiresAt: future})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newService(t)

			_, err := svc.ParseToken(tt.token(t))
			checkErr(t, err, models.ErrUnauthorized, "")
		})
	}
}

func TestCreateTask(t *testing.T) {
	creator, other := uuid.New(), uuid.New()
	deadline := time.Now().Add(24 * time.Hour)

	// expectCreate asserts the task passed to the repository and echoes it back.
	expectCreate := func(t *testing.T, d deps, wantAssignee uuid.UUID) {
		d.tasks.EXPECT().CreateTask(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, task models.Task) (models.Task, error) {
				if task.CreatorID != creator || task.AssigneeID != wantAssignee {
					t.Errorf("creator/assignee = %s/%s, want %s/%s", task.CreatorID, task.AssigneeID, creator, wantAssignee)
				}

				if task.Status != models.StatusCreated || task.Title != "Write docs" {
					t.Errorf("status/title = %q/%q, want created/\"Write docs\"", task.Status, task.Title)
				}

				task.ID = uuid.New()

				return task, nil
			})
	}

	tests := []struct {
		name      string
		in        models.NewTask
		setup     func(t *testing.T, d deps)
		wantField string
		wantErr   error
	}{
		{
			name:  "assigned to creator by default",
			in:    models.NewTask{Title: "  Write docs ", Deadline: deadline},
			setup: func(t *testing.T, d deps) { expectCreate(t, d, creator) },
		},
		{
			name: "assigned to another user",
			in:   models.NewTask{Title: "Write docs", Deadline: deadline, AssigneeID: &other},
			setup: func(t *testing.T, d deps) {
				d.users.EXPECT().GetUserByID(gomock.Any(), other).Return(models.User{ID: other}, nil)
				expectCreate(t, d, other)
			},
		},
		{
			name: "unknown assignee",
			in:   models.NewTask{Title: "Write docs", Deadline: deadline, AssigneeID: &other},
			setup: func(_ *testing.T, d deps) {
				d.users.EXPECT().GetUserByID(gomock.Any(), other).Return(models.User{}, models.ErrNotFound)
			},
			wantField: "assignee_id",
		},
		{
			name: "assignee lookup fails",
			in:   models.NewTask{Title: "Write docs", Deadline: deadline, AssigneeID: &other},
			setup: func(_ *testing.T, d deps) {
				d.users.EXPECT().GetUserByID(gomock.Any(), other).Return(models.User{}, errDB)
			},
			wantErr: errDB,
		},
		{name: "empty title", in: models.NewTask{Title: "   ", Deadline: deadline}, wantField: "title"},
		{name: "too long title", in: models.NewTask{Title: strings.Repeat("a", 201), Deadline: deadline}, wantField: "title"},
		{name: "missing deadline", in: models.NewTask{Title: "Write docs"}, wantField: "deadline"},
		{
			name:      "deadline in the past",
			in:        models.NewTask{Title: "Write docs", Deadline: time.Now().Add(-time.Minute)},
			wantField: "deadline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newService(t)
			if tt.setup != nil {
				tt.setup(t, d)
			}

			_, err := svc.CreateTask(t.Context(), creator, tt.in)
			checkErr(t, err, tt.wantErr, tt.wantField)
		})
	}
}

func TestUpdateTaskPermissions(t *testing.T) {
	creator, assignee, stranger := uuid.New(), uuid.New(), uuid.New()
	task := models.Task{ID: uuid.New(), CreatorID: creator, AssigneeID: assignee, Status: models.StatusCreated}

	inProgress := models.StatusInProgress
	bogus := models.Status("archived")
	title := "New title"

	tests := []struct {
		name       string
		userID     uuid.UUID
		upd        models.TaskUpdate
		getErr     error
		setup      func(d deps)
		wantUpdate bool
		wantField  string
		wantErr    error
	}{
		{name: "creator changes title", userID: creator, upd: models.TaskUpdate{Title: &title}, wantUpdate: true},
		{name: "creator changes status", userID: creator, upd: models.TaskUpdate{Status: &inProgress}, wantUpdate: true},
		{
			name:   "creator reassigns task",
			userID: creator,
			upd:    models.TaskUpdate{AssigneeID: &stranger},
			setup: func(d deps) {
				d.users.EXPECT().GetUserByID(gomock.Any(), stranger).Return(models.User{ID: stranger}, nil)
			},
			wantUpdate: true,
		},
		{
			name:   "creator reassigns to unknown user",
			userID: creator,
			upd:    models.TaskUpdate{AssigneeID: &stranger},
			setup: func(d deps) {
				d.users.EXPECT().GetUserByID(gomock.Any(), stranger).Return(models.User{}, models.ErrNotFound)
			},
			wantField: "assignee_id",
		},
		{name: "assignee changes status", userID: assignee, upd: models.TaskUpdate{Status: &inProgress}, wantUpdate: true},
		{
			name:    "assignee changes title",
			userID:  assignee,
			upd:     models.TaskUpdate{Title: &title},
			wantErr: models.ErrForbidden,
		},
		{
			name:    "assignee changes status and title",
			userID:  assignee,
			upd:     models.TaskUpdate{Status: &inProgress, Title: &title},
			wantErr: models.ErrForbidden,
		},
		{name: "stranger changes status", userID: stranger, upd: models.TaskUpdate{Status: &inProgress}, wantErr: models.ErrForbidden},
		{
			name:    "task not found",
			userID:  creator,
			upd:     models.TaskUpdate{Title: &title},
			getErr:  models.ErrNotFound,
			wantErr: models.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newService(t)
			d.tasks.EXPECT().GetTask(gomock.Any(), task.ID).Return(task, tt.getErr)

			if tt.setup != nil {
				tt.setup(d)
			}

			if tt.wantUpdate {
				d.tasks.EXPECT().UpdateTask(gomock.Any(), task.ID, tt.upd).Return(task, nil)
			}

			_, err := svc.UpdateTask(t.Context(), tt.userID, task.ID, tt.upd)
			checkErr(t, err, tt.wantErr, tt.wantField)
		})
	}

	t.Run("invalid input is rejected before loading the task", func(t *testing.T) {
		cases := []struct {
			upd   models.TaskUpdate
			field string
		}{
			{upd: models.TaskUpdate{}, field: "body"},
			{upd: models.TaskUpdate{Status: &bogus}, field: "status"},
		}

		for _, c := range cases {
			svc, _ := newService(t)

			_, err := svc.UpdateTask(t.Context(), creator, task.ID, c.upd)
			checkErr(t, err, nil, c.field)
		}
	})
}

func TestDeleteTask(t *testing.T) {
	creator, assignee := uuid.New(), uuid.New()
	task := models.Task{ID: uuid.New(), CreatorID: creator, AssigneeID: assignee}

	tests := []struct {
		name       string
		userID     uuid.UUID
		getErr     error
		wantDelete bool
		wantErr    error
	}{
		{name: "creator deletes", userID: creator, wantDelete: true},
		{name: "assignee is forbidden", userID: assignee, wantErr: models.ErrForbidden},
		{name: "stranger is forbidden", userID: uuid.New(), wantErr: models.ErrForbidden},
		{name: "task not found", userID: creator, getErr: models.ErrNotFound, wantErr: models.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newService(t)
			d.tasks.EXPECT().GetTask(gomock.Any(), task.ID).Return(task, tt.getErr)

			if tt.wantDelete {
				d.tasks.EXPECT().DeleteTask(gomock.Any(), task.ID).Return(nil)
			}

			err := svc.DeleteTask(t.Context(), tt.userID, task.ID)
			checkErr(t, err, tt.wantErr, "")
		})
	}
}

func TestListTasks(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	bogus := models.Status("archived")

	tests := []struct {
		name      string
		filter    models.TaskFilter
		wantLimit int
		repoErr   error
		wantField string
		wantErr   error
	}{
		{name: "default limit", filter: models.TaskFilter{}, wantLimit: 20},
		{name: "explicit limit and offset", filter: models.TaskFilter{Limit: 5, Offset: 10}, wantLimit: 5},
		{name: "repository failure", filter: models.TaskFilter{}, wantLimit: 20, repoErr: errDB, wantErr: errDB},
		{name: "limit above maximum", filter: models.TaskFilter{Limit: 101}, wantField: "limit"},
		{name: "negative limit", filter: models.TaskFilter{Limit: -1}, wantField: "limit"},
		{name: "negative offset", filter: models.TaskFilter{Offset: -1}, wantField: "offset"},
		{name: "unknown status", filter: models.TaskFilter{Status: &bogus}, wantField: "status"},
		{
			name:      "deadline range reversed",
			filter:    models.TaskFilter{DeadlineFrom: &later, DeadlineTo: &now},
			wantField: "deadline_from",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newService(t)

			if tt.wantLimit > 0 {
				want := tt.filter
				want.Limit = tt.wantLimit
				d.tasks.EXPECT().ListTasks(gomock.Any(), want).Return([]models.Task{{ID: uuid.New()}}, 42, tt.repoErr)
			}

			page, err := svc.ListTasks(t.Context(), tt.filter)
			checkErr(t, err, tt.wantErr, tt.wantField)

			if tt.wantLimit > 0 && tt.wantErr == nil {
				if page.Total != 42 || page.Limit != tt.wantLimit || page.Offset != tt.filter.Offset || len(page.Items) != 1 {
					t.Errorf("unexpected page: %+v", page)
				}
			}
		})
	}
}
