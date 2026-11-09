package handler

import (
	"net/http"
)

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)

		return
	}

	user, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusCreated, toUserResponse(user))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)

		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	h.writeJSON(w, r, http.StatusOK, tokenResponse{
		AccessToken: token.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(token.ExpiresIn.Seconds()),
	})
}
