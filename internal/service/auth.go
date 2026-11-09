package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/mmfshirokan/task-manager/models"
)

const (
	minPasswordLength = 8
	// bcrypt ignores input beyond 72 bytes.
	maxPasswordBytes = 72
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

	user, err := s.users.CreateUser(ctx, email, string(hash))
	if err != nil {
		return models.User{}, err
	}

	s.log.InfoContext(ctx, "user registered", "user_id", user.ID)

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (models.Token, error) {
	email = normalizeEmail(email)

	user, err := s.users.GetUserByEmail(ctx, email)
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

// ParseToken validates a bearer access token and returns the user ID it was
// issued for.
func (s *Service) ParseToken(raw string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return s.cfg.JWTSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		s.log.Debug("token rejected", "error", err)

		return uuid.Nil, fmt.Errorf("%w: invalid or expired token", models.ErrUnauthorized)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid token subject", models.ErrUnauthorized)
	}

	return userID, nil
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
