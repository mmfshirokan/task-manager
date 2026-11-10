package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/mmfshirokan/task-manager/models"
)

const maxBodyBytes = 1 << 20

type Writer struct {
	log *slog.Logger
}

func NewWriter(log *slog.Logger) *Writer {
	return &Writer{log: log}
}

func (rw *Writer) JSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		rw.log.ErrorContext(r.Context(), "write response", "error", err)
	}
}

// Error сопоставляет доменные ошибки с HTTP-статусами. Неизвестные ошибки
// логируются, а клиент получает общий ответ 500 без подробностей.
func (rw *Writer) Error(w http.ResponseWriter, r *http.Request, err error) {
	status := statusFor(err)
	message := err.Error()

	if status == http.StatusInternalServerError {
		rw.log.ErrorContext(r.Context(), "internal error",
			"error", err, "request_id", chimiddleware.GetReqID(r.Context()))

		message = "internal server error"
	}

	rw.JSON(w, r, status, models.ErrorResponse{Error: message})
}

// DecodeJSON декодирует тело запроса в dst с ограничением размера и отклоняет
// неизвестные поля.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return &models.ValidationError{Field: "body", Message: "invalid JSON: " + err.Error()}
	}

	return nil
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
