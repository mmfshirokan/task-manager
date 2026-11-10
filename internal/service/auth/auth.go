package auth

//go:generate go tool mockgen -source=auth.go -destination=mocks/auth_mock.go -package=mocks

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/mmfshirokan/task-manager/models"
)

const (
	minPasswordLength = 8
	// bcrypt игнорирует всё, что длиннее 72 байт.
	maxPasswordBytes = 72
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

type Config struct {
	JWTSecret []byte
	TokenTTL  time.Duration
}

type Service struct {
	users UserRepository
	cfg   Config
	log   *slog.Logger
}

func New(users UserRepository, cfg Config, log *slog.Logger) *Service {
	return &Service{users: users, cfg: cfg, log: log}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentials(email, password string) error {
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return &models.ValidationError{Field: "email", Message: "must be a valid email address"}
	}

	switch {
	case len(password) < minPasswordLength:
		return &models.ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("must be at least %d characters", minPasswordLength),
		}
	case len(password) > maxPasswordBytes:
		return &models.ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("must be at most %d bytes", maxPasswordBytes),
		}
	}

	return nil
}
