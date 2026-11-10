package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/mmfshirokan/task-manager/models"
)

func (s *Service) Login(ctx context.Context, email, password string) (models.Token, error) {
	email = normalizeEmail(email)

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.log.WarnContext(ctx, "login failed: unknown email")

			return models.Token{}, models.ErrInvalidCredentials
		}

		return models.Token{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.log.WarnContext(ctx, "login failed: wrong password", "user_id", user.ID)

		return models.Token{}, models.ErrInvalidCredentials
	}

	token, err := s.issueToken(user.ID)
	if err != nil {
		return models.Token{}, err
	}

	s.log.InfoContext(ctx, "user logged in", "user_id", user.ID)

	return token, nil
}

func (s *Service) issueToken(userID uuid.UUID) (models.Token, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.TokenTTL)),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.cfg.JWTSecret)
	if err != nil {
		return models.Token{}, fmt.Errorf("sign token: %w", err)
	}

	return models.Token{AccessToken: signed, ExpiresIn: s.cfg.TokenTTL}, nil
}
