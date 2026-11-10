package auth

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/mmfshirokan/task-manager/models"
)

func (s *Service) Register(ctx context.Context, email, password string) (models.User, error) {
	email = normalizeEmail(email)

	if err := validateCredentials(email, password); err != nil {
		return models.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, email, string(hash))
	if err != nil {
		return models.User{}, err
	}

	s.log.InfoContext(ctx, "user registered", "user_id", user.ID)

	return user, nil
}
