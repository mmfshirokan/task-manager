package handler

import (
	"net/http"
)

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDParam(r)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	if err := h.tasks.DeleteTask(r.Context(), userIDFrom(r.Context()), id); err != nil {
		h.writeError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
