package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

type userIDKey struct{}

// authenticate requires a valid "Authorization: Bearer <token>" header and
// stores the user ID in the request context.
func (h *Handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			h.writeError(w, r, fmt.Errorf("%w: missing or malformed bearer token", models.ErrUnauthorized))

			return
		}

		userID, err := h.auth.ParseToken(token)
		if err != nil {
			h.writeError(w, r, err)

			return
		}

		ctx := context.WithValue(r.Context(), userIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		h.log.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration", time.Since(start).String(),
			"request_id", chimiddleware.GetReqID(r.Context()),
		)
	})
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	token = strings.TrimSpace(token)

	return token, token != ""
}

func userIDFrom(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey{}).(uuid.UUID)

	return id
}
