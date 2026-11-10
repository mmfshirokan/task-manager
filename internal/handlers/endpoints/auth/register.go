package auth

import (
	"net/http"

	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

// @Summary		Регистрация пользователя
// @Description	Создаёт нового пользователя. Email приводится к нижнему регистру и должен быть уникальным, пароль — от 8 до 72 байт.
// @Tags			Авторизация
// @Accept			json
// @Produce		json
// @Param			request	body		models.CredentialsRequest	true	"Email и пароль"
// @Success		201		{object}	models.UserResponse			"Пользователь создан"
// @Failure		400		{object}	models.ErrorResponse		"Некорректный email, пароль или JSON"
// @Failure		409		{object}	models.ErrorResponse		"Пользователь с таким email уже существует"
// @Failure		500		{object}	models.ErrorResponse		"Внутренняя ошибка сервера"
// @Router			/api/v1/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CredentialsRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		h.resp.Error(w, r, err)

		return
	}

	user, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	h.resp.JSON(w, r, http.StatusCreated, models.NewUserResponse(user))
}
