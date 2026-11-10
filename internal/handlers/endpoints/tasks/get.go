package tasks

import (
	"net/http"

	"github.com/mmfshirokan/task-manager/models"
)

// @Summary		Получение задачи
// @Description	Возвращает задачу по идентификатору. Доступно любому авторизованному пользователю.
// @Tags			Задачи
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string					true	"Идентификатор задачи"	Format(uuid)
// @Success		200	{object}	models.TaskResponse		"Задача"
// @Failure		400	{object}	models.ErrorResponse	"Некорректный идентификатор"
// @Failure		401	{object}	models.ErrorResponse	"Нет токена или токен недействителен"
// @Failure		404	{object}	models.ErrorResponse	"Задача не найдена"
// @Failure		500	{object}	models.ErrorResponse	"Внутренняя ошибка сервера"
// @Router			/api/v1/tasks/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDParam(r)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	task, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	h.resp.JSON(w, r, http.StatusOK, models.NewTaskResponse(task))
}
