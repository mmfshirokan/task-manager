package handler

import (
	"net/http"
)

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)

		return
	}

	task, err := h.tasks.CreateTask(r.Context(), userIDFrom(r.Context()), req.toModel())
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusCreated, toTaskResponse(task))
}
