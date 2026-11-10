package tasks

import (
	"net/http"

	"github.com/mmfshirokan/task-manager/internal/handlers/middleware"
	"github.com/mmfshirokan/task-manager/internal/handlers/response"
	"github.com/mmfshirokan/task-manager/models"
)

// @Summary		Создание задачи
// @Description	Создаёт задачу от имени текущего пользователя со статусом created. Если исполнитель не указан, им становится автор.
// @Tags			Задачи
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		models.CreateTaskRequest	true	"Данные задачи"
// @Success		201		{object}	models.TaskResponse			"Задача создана"
// @Failure		400		{object}	models.ErrorResponse		"Ошибка валидации: пустое название, срок в прошлом, несуществующий исполнитель"
// @Failure		401		{object}	models.ErrorResponse		"Нет токена или токен недействителен"
// @Failure		500		{object}	models.ErrorResponse		"Внутренняя ошибка сервера"
// @Router			/api/v1/tasks [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		h.resp.Error(w, r, err)

		return
	}

	task, err := h.svc.Create(r.Context(), middleware.UserID(r.Context()), req.ToNewTask())
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	h.resp.JSON(w, r, http.StatusCreated, models.NewTaskResponse(task))
}
