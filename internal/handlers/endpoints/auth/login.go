package auth

import (
	"net/http"

	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

// @Summary		Вход в систему
// @Description	Проверяет email и пароль и выдаёт JWT-токен доступа. Токен передаётся в заголовке Authorization: Bearer <токен>.
// @Tags			Авторизация
// @Accept			json
// @Produce		json
// @Param			request	body		models.CredentialsRequest	true	"Email и пароль"
// @Success		200		{object}	models.TokenResponse		"Токен выдан"
// @Failure		400		{object}	models.ErrorResponse		"Некорректный JSON"
// @Failure		401		{object}	models.ErrorResponse		"Неверный email или пароль"
// @Failure		500		{object}	models.ErrorResponse		"Внутренняя ошибка сервера"
// @Router			/api/v1/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.CredentialsRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		h.resp.Error(w, r, err)

		return
	}

	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	h.resp.JSON(w, r, http.StatusOK, models.NewTokenResponse(token))
}
