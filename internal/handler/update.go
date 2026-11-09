package handler

import (
	"net/http"
)

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDParam(r)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	var req updateTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)

		return
	}

	task, err := h.tasks.UpdateTask(r.Context(), userIDFrom(r.Context()), id, req.toModel())
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, toTaskResponse(task))
}
