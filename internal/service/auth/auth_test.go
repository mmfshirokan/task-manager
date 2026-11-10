package auth_test

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

	"github.com/mmfshirokan/task-manager/internal/service/auth"
	"github.com/mmfshirokan/task-manager/internal/service/auth/mocks"
	"github.com/mmfshirokan/task-manager/models"
)

var (
	testSecret = []byte("test-secret")
	errDB      = errors.New("db is down")
)

func newService(t *testing.T) (*auth.Service, *mocks.MockUserRepository) {
	t.Helper()

	users := mocks.NewMockUserRepository(gomock.NewController(t))
	cfg := auth.Config{JWTSecret: testSecret, TokenTTL: time.Hour}

	return auth.New(users, cfg, slog.New(slog.DiscardHandler)), users
}

// checkErr проверяет результат: ошибку валидации поля wantField, ошибку wantErr
// или отсутствие ошибки.
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
		setup     func(t *testing.T, users *mocks.MockUserRepository)
		wantField string
		wantErr   error
	}{
		{
			name:     "normalizes email and hashes password",
			email:    "  Alice@Example.COM ",
			password: "secret123",
			setup: func(t *testing.T, users *mocks.MockUserRepository) {
				users.EXPECT().Create(gomock.Any(), "alice@example.com", gomock.Any()).
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
			setup: func(_ *testing.T, users *mocks.MockUserRepository) {
				users.EXPECT().Create(gomock.Any(), "bob@example.com", gomock.Any()).
					Return(models.User{}, fmt.Errorf("user: %w", models.ErrAlreadyExists))
			},
			wantErr: models.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, users := newService(t)
			if tt.setup != nil {
				tt.setup(t, users)
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
			svc, users := newService(t)
			users.EXPECT().GetByEmail(gomock.Any(), "alice@example.com").Return(tt.repoUser, tt.repoErr)

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
