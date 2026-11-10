package tasks

import (
	"net/http"

	"github.com/mmfshirokan/task-manager/internal/handlers/middleware"
	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

// @Summary		Изменение задачи
// @Description	Частично обновляет задачу: изменяются только переданные поля. Автор может менять любые поля, исполнитель — только статус.
// @Tags			Задачи
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Идентификатор задачи"	Format(uuid)
// @Param			request	body		models.UpdateTaskRequest	true	"Изменяемые поля"
// @Success		200		{object}	models.TaskResponse			"Задача обновлена"
// @Failure		400		{object}	models.ErrorResponse		"Ошибка валидации или пустой запрос"
// @Failure		401		{object}	models.ErrorResponse		"Нет токена или токен недействителен"
// @Failure		403		{object}	models.ErrorResponse		"Нет прав на изменение этих полей"
// @Failure		404		{object}	models.ErrorResponse		"Задача не найдена"
// @Failure		500		{object}	models.ErrorResponse		"Внутренняя ошибка сервера"
// @Router			/api/v1/tasks/{id} [patch]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDParam(r)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	var req models.UpdateTaskRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		h.resp.Error(w, r, err)

		return
	}

	task, err := h.svc.Update(r.Context(), middleware.UserID(r.Context()), id, req.ToTaskUpdate())
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	h.resp.JSON(w, r, http.StatusOK, models.NewTaskResponse(task))
}
