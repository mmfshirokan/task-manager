package repository_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/mmfshirokan/task-manager/internal/repository"
	"github.com/mmfshirokan/task-manager/models"
	"github.com/mmfshirokan/task-manager/schema"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		os.Exit(m.Run())
	}

	code, err := runWithPostgres(m)
	if err != nil {
		log.Printf("integration tests: %v", err)

		code = 1
	}

	os.Exit(code)
}

// runWithPostgres поднимает временный контейнер Postgres, применяет миграции и
// запускает на нём тесты.
func runWithPostgres(m *testing.M) (int, error) {
	dockerPool, err := dockertest.NewPool("")
	if err != nil {
		return 0, fmt.Errorf("connect to docker: %w", err)
	}

	dockerPool.MaxWait = time.Minute

	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "17-alpine",
		Env:        []string{"POSTGRES_USER=test", "POSTGRES_PASSWORD=test", "POSTGRES_DB=tasks"},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return 0, fmt.Errorf("start postgres container: %w", err)
	}

	defer func() {
		if err := dockerPool.Purge(resource); err != nil {
			log.Printf("purge postgres container: %v", err)
		}
	}()

	// Страховка: docker удалит контейнер, даже если тестовый процесс будет убит.
	if err := resource.Expire(300); err != nil {
		return 0, fmt.Errorf("set container expiry: %w", err)
	}

	ctx := context.Background()
	dsn := fmt.Sprintf("postgres://test:test@%s/tasks?sslmode=disable", resource.GetHostPort("5432/tcp"))

	if err := dockerPool.Retry(func() error { return connect(ctx, dsn) }); err != nil {
		return 0, fmt.Errorf("wait for postgres: %w", err)
	}

	defer testPool.Close()

	if _, err := schema.Migrate(ctx, dsn); err != nil {
		return 0, err
	}

	return m.Run(), nil
}

func connect(ctx context.Context, dsn string) error {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()

		return err
	}

	testPool = db

	return nil
}

// newRepo пропускает тест в режиме -short и возвращает репозиторий с пустыми таблицами.
func newRepo(t *testing.T) *repository.Repository {
	t.Helper()

	if testing.Short() {
		t.Skip("integration test: requires Docker, skipped with -short")
	}

	if _, err := testPool.Exec(t.Context(), `TRUNCATE tasks, users CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	return repository.New(testPool)
}

func mustCreateUser(t *testing.T, repo *repository.Repository, email string) models.User {
	t.Helper()

	user, err := repo.Users.Create(t.Context(), email, "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	return user
}

func mustCreateTask(t *testing.T, repo *repository.Repository, task models.Task) models.Task {
	t.Helper()

	if task.Status == "" {
		task.Status = models.StatusCreated
	}

	created, err := repo.Tasks.Create(t.Context(), task)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	return created
}

func mustGetTask(t *testing.T, repo *repository.Repository, id uuid.UUID) models.Task {
	t.Helper()

	task, err := repo.Tasks.Get(t.Context(), id)
	if err != nil {
		t.Fatalf("get task %s: %v", id, err)
	}

	return task
}

func wantValidationErr(t *testing.T, err error, field string) {
	t.Helper()

	var validationErr *models.ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != field {
		t.Fatalf("want validation error on %q, got %v", field, err)
	}
}

func TestUsersCreate(t *testing.T) {
	repo := newRepo(t)

	alice := mustCreateUser(t, repo, "alice@example.com")
	if alice.ID == uuid.Nil || alice.CreatedAt.IsZero() || alice.Email != "alice@example.com" {
		t.Errorf("generated columns are not returned: %+v", alice)
	}

	if _, err := repo.Users.Create(t.Context(), "alice@example.com", "other"); !errors.Is(err, models.ErrAlreadyExists) {
		t.Errorf("duplicate email: want ErrAlreadyExists, got %v", err)
	}
}

func TestUsersGetByEmail(t *testing.T) {
	repo := newRepo(t)
	alice := mustCreateUser(t, repo, "alice@example.com")

	got, err := repo.Users.GetByEmail(t.Context(), "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}

	if got.ID != alice.ID || got.PasswordHash != "hash" {
		t.Errorf("GetByEmail = %+v, want %+v", got, alice)
	}

	if _, err := repo.Users.GetByEmail(t.Context(), "nobody@example.com"); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("unknown email: want ErrNotFound, got %v", err)
	}
}

func TestUsersGetByID(t *testing.T) {
	repo := newRepo(t)
	alice := mustCreateUser(t, repo, "alice@example.com")

	got, err := repo.Users.GetByID(t.Context(), alice.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Email != alice.Email {
		t.Errorf("GetByID = %+v, want %+v", got, alice)
	}

	if _, err := repo.Users.GetByID(t.Context(), uuid.New()); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("unknown id: want ErrNotFound, got %v", err)
	}
}

func TestTasksLifecycle(t *testing.T) {
	repo := newRepo(t)
	ctx := t.Context()
	alice := mustCreateUser(t, repo, "alice@example.com")
	bob := mustCreateUser(t, repo, "bob@example.com")
	deadline := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Microsecond)

	task := mustCreateTask(t, repo, models.Task{
		Title: "Write docs", Description: "API reference", Deadline: deadline,
		CreatorID: alice.ID, AssigneeID: alice.ID,
	})
	if task.ID == uuid.Nil || task.Status != models.StatusCreated || !task.Deadline.Equal(deadline) {
		t.Fatalf("unexpected created task: %+v", task)
	}

	if got := mustGetTask(t, repo, task.ID); got.Title != "Write docs" || got.Description != "API reference" {
		t.Fatalf("Get = %+v", got)
	}

	title, status := "Write better docs", models.StatusInProgress

	updated, err := repo.Tasks.Update(ctx, task.ID, models.TaskUpdate{Title: &title, Status: &status, AssigneeID: &bob.ID})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Title != title || updated.Status != status || updated.AssigneeID != bob.ID ||
		updated.Description != "API reference" || !updated.Deadline.Equal(deadline) {
		t.Errorf("partial update changed the wrong fields: %+v", updated)
	}

	if !updated.UpdatedAt.After(task.UpdatedAt) {
		t.Errorf("updated_at was not bumped: %s -> %s", task.UpdatedAt, updated.UpdatedAt)
	}

	if err := repo.Tasks.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := repo.Tasks.Get(ctx, task.ID); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("get deleted task: want ErrNotFound, got %v", err)
	}
}

func TestTasksErrors(t *testing.T) {
	repo := newRepo(t)
	ctx := t.Context()
	alice := mustCreateUser(t, repo, "alice@example.com")
	task := mustCreateTask(t, repo, models.Task{
		Title: "Task", Deadline: time.Now().Add(time.Hour), CreatorID: alice.ID, AssigneeID: alice.ID,
	})
	unknown := uuid.New()
	title := "x"

	_, err := repo.Tasks.Create(ctx, models.Task{
		Title: "Task", Deadline: time.Now(), Status: models.StatusCreated, CreatorID: alice.ID, AssigneeID: unknown,
	})
	wantValidationErr(t, err, "assignee_id")

	_, err = repo.Tasks.Update(ctx, task.ID, models.TaskUpdate{AssigneeID: &unknown})
	wantValidationErr(t, err, "assignee_id")

	if _, err := repo.Tasks.Update(ctx, unknown, models.TaskUpdate{Title: &title}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("update unknown task: want ErrNotFound, got %v", err)
	}

	if err := repo.Tasks.Delete(ctx, unknown); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("delete unknown task: want ErrNotFound, got %v", err)
	}
}

func TestTasksStatusEnumRoundTrip(t *testing.T) {
	repo := newRepo(t)
	alice := mustCreateUser(t, repo, "alice@example.com")

	for _, status := range []models.Status{
		models.StatusCreated, models.StatusInProgress, models.StatusCompleted, models.StatusOverdue,
	} {
		task := mustCreateTask(t, repo, models.Task{
			Title: string(status), Deadline: time.Now().Add(time.Hour), Status: status,
			CreatorID: alice.ID, AssigneeID: alice.ID,
		})

		if got := mustGetTask(t, repo, task.ID); got.Status != status {
			t.Errorf("status %q round trip = %q", status, got.Status)
		}
	}

	bogus := models.Status("archived")
	if _, _, err := repo.Tasks.List(t.Context(), models.TaskFilter{Status: &bogus, Limit: 10}); err == nil {
		t.Error("value outside the enum must be rejected by Postgres")
	}
}

func TestTasksList(t *testing.T) {
	repo := newRepo(t)
	alice := mustCreateUser(t, repo, "alice@example.com")
	bob := mustCreateUser(t, repo, "bob@example.com")
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	seed := []struct {
		creator  uuid.UUID
		assignee uuid.UUID
		status   models.Status
		days     int
	}{
		{alice.ID, alice.ID, models.StatusCreated, 1},
		{alice.ID, alice.ID, models.StatusInProgress, 2},
		{alice.ID, bob.ID, models.StatusInProgress, 3},
		{bob.ID, bob.ID, models.StatusCompleted, 4},
		{bob.ID, bob.ID, models.StatusCreated, 5},
	}
	for i, s := range seed {
		mustCreateTask(t, repo, models.Task{
			Title: fmt.Sprintf("task %d", i+1), Deadline: base.AddDate(0, 0, s.days), Status: s.status,
			CreatorID: s.creator, AssigneeID: s.assignee,
		})
	}

	inProgress := models.StatusInProgress
	from, to := base.AddDate(0, 0, 2), base.AddDate(0, 0, 4)

	tests := []struct {
		name       string
		filter     models.TaskFilter
		wantTotal  int
		wantTitles []string
	}{
		{
			name:       "no filter, newest first",
			filter:     models.TaskFilter{Limit: 10},
			wantTotal:  5,
			wantTitles: []string{"task 5", "task 4", "task 3", "task 2", "task 1"},
		},
		{
			name:       "by status",
			filter:     models.TaskFilter{Status: &inProgress, Limit: 10},
			wantTotal:  2,
			wantTitles: []string{"task 3", "task 2"},
		},
		{
			name:       "by assignee",
			filter:     models.TaskFilter{AssigneeID: &alice.ID, Limit: 10},
			wantTotal:  2,
			wantTitles: []string{"task 2", "task 1"},
		},
		{
			name:       "by creator",
			filter:     models.TaskFilter{CreatorID: &bob.ID, Limit: 10},
			wantTotal:  2,
			wantTitles: []string{"task 5", "task 4"},
		},
		{
			name:       "by creator and assignee",
			filter:     models.TaskFilter{CreatorID: &alice.ID, AssigneeID: &bob.ID, Limit: 10},
			wantTotal:  1,
			wantTitles: []string{"task 3"},
		},
		{
			name:       "by deadline range (inclusive)",
			filter:     models.TaskFilter{DeadlineFrom: &from, DeadlineTo: &to, Limit: 10},
			wantTotal:  3,
			wantTitles: []string{"task 4", "task 3", "task 2"},
		},
		{
			name:       "combined filters",
			filter:     models.TaskFilter{Status: &inProgress, AssigneeID: &bob.ID, Limit: 10},
			wantTotal:  1,
			wantTitles: []string{"task 3"},
		},
		{
			name:       "pagination keeps the total",
			filter:     models.TaskFilter{Limit: 2, Offset: 2},
			wantTotal:  5,
			wantTitles: []string{"task 3", "task 2"},
		},
		{
			name:       "offset past the end",
			filter:     models.TaskFilter{Limit: 2, Offset: 10},
			wantTotal:  5,
			wantTitles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, total, err := repo.Tasks.List(t.Context(), tt.filter)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			titles := make([]string, 0, len(list))
			for _, task := range list {
				titles = append(titles, task.Title)
			}

			if total != tt.wantTotal || fmt.Sprint(titles) != fmt.Sprint(tt.wantTitles) {
				t.Errorf("got total %d, titles %v; want %d, %v", total, titles, tt.wantTotal, tt.wantTitles)
			}
		})
	}
}

func TestTasksMarkOverdue(t *testing.T) {
	repo := newRepo(t)
	alice := mustCreateUser(t, repo, "alice@example.com")
	past, future := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)

	seed := []struct {
		deadline time.Time
		status   models.Status
		want     models.Status
	}{
		{past, models.StatusCreated, models.StatusOverdue},
		{past, models.StatusInProgress, models.StatusOverdue},
		{past, models.StatusCompleted, models.StatusCompleted},
		{future, models.StatusCreated, models.StatusCreated},
		{future, models.StatusInProgress, models.StatusInProgress},
	}

	ids := make([]uuid.UUID, len(seed))
	for i, s := range seed {
		ids[i] = mustCreateTask(t, repo, models.Task{
			Title: "task", Deadline: s.deadline, Status: s.status, CreatorID: alice.ID, AssigneeID: alice.ID,
		}).ID
	}

	marked, err := repo.Tasks.MarkOverdue(t.Context())
	if err != nil {
		t.Fatalf("MarkOverdue: %v", err)
	}

	if marked != 2 {
		t.Fatalf("MarkOverdue = %d, want 2", marked)
	}

	for i, s := range seed {
		if got := mustGetTask(t, repo, ids[i]); got.Status != s.want {
			t.Errorf("task %d (%s): status %q, want %q", i, s.status, got.Status, s.want)
		}
	}

	marked, err = repo.Tasks.MarkOverdue(t.Context())
	if err != nil {
		t.Fatalf("second MarkOverdue: %v", err)
	}

	if marked != 0 {
		t.Errorf("second MarkOverdue = %d, want 0", marked)
	}
}
