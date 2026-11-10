package auth

import (
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

// ParseToken проверяет токен доступа и возвращает идентификатор пользователя,
// которому он выдан.
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
