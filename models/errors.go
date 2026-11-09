package models

import (
	"errors"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

// ValidationError reports invalid client input for a single field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
