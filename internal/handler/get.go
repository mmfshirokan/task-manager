package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDParam(r)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	task, err := h.tasks.GetTask(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, toTaskResponse(task))
}

func taskIDParam(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, &models.ValidationError{Field: "id", Message: "must be a valid UUID"}
	}

	return id, nil
}
