package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

type TokenParser interface {
	ParseToken(token string) (uuid.UUID, error)
}

type userIDKey struct{}

// Authenticate требует корректный заголовок "Authorization: Bearer <токен>" и
// кладёт идентификатор пользователя в контекст запроса.
func Authenticate(parser TokenParser, resp *response.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				resp.Error(w, r, fmt.Errorf("%w: missing or malformed bearer token", models.ErrUnauthorized))

				return
			}

			userID, err := parser.ParseToken(token)
			if err != nil {
				resp.Error(w, r, err)

				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey{}).(uuid.UUID)

	return id
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	token = strings.TrimSpace(token)

	return token, token != ""
}
