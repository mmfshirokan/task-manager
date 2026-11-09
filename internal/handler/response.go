package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/mmfshirokan/task-manager/models"
)

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.log.ErrorContext(r.Context(), "write response", "error", err)
	}
}

// writeError maps domain errors to HTTP statuses. Unknown errors are logged
// and hidden from the client behind a generic 500.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := statusFor(err)
	message := err.Error()

	if status == http.StatusInternalServerError {
		h.log.ErrorContext(r.Context(), "internal error",
			"error", err, "request_id", chimiddleware.GetReqID(r.Context()))

		message = "internal server error"
	}

	h.writeJSON(w, r, status, errorResponse{Error: message})
}

func statusFor(err error) int {
	var validationErr *models.ValidationError

	switch {
	case errors.As(err, &validationErr):
		return http.StatusBadRequest
	case errors.Is(err, models.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, models.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, models.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, models.ErrInvalidCredentials), errors.Is(err, models.ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
