package repository

import (
	"github.com/mmfshirokan/task-manager/internal/repository/tasks"
	"github.com/mmfshirokan/task-manager/internal/repository/users"
)

type DB interface {
	users.DB
	tasks.DB
}

type Repository struct {
	Users *users.Repository
	Tasks *tasks.Repository
}

func New(db DB) *Repository {
	return &Repository{
		Users: users.New(db),
		Tasks: tasks.New(db),
	}
}
