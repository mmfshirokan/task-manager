package tasks

import (
	"net/http"

	"github.com/mmfshirokan/task-manager/internal/handlers/middleware"
)

// @Summary		Удаление задачи
// @Description	Удаляет задачу. Удалить задачу может только её автор.
// @Tags			Задачи
// @Produce		json
// @Security		BearerAuth
// @Param			id	path	string	true	"Идентификатор задачи"	Format(uuid)
// @Success		204	"Задача удалена"
// @Failure		400	{object}	models.ErrorResponse	"Некорректный идентификатор"
// @Failure		401	{object}	models.ErrorResponse	"Нет токена или токен недействителен"
// @Failure		403	{object}	models.ErrorResponse	"Удалить задачу может только автор"
// @Failure		404	{object}	models.ErrorResponse	"Задача не найдена"
// @Failure		500	{object}	models.ErrorResponse	"Внутренняя ошибка сервера"
// @Router			/api/v1/tasks/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := taskIDParam(r)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	if err := h.svc.Delete(r.Context(), middleware.UserID(r.Context()), id); err != nil {
		h.resp.Error(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
