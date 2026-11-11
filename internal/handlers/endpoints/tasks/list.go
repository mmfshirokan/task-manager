package tasks

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

// @Summary		Список задач
// @Description	Возвращает страницу задач всех пользователей, от новых к старым. Поддерживает фильтры по статусу, исполнителю, автору и диапазону сроков.
// @Tags			Задачи
// @Produce		json
// @Security		BearerAuth
// @Param			status			query		string					false	"Статус задачи"								Enums(created, in_progress, completed, overdue)
// @Param			assignee_id		query		string					false	"Идентификатор исполнителя"					Format(uuid)
// @Param			creator_id		query		string					false	"Идентификатор автора"						Format(uuid)
// @Param			deadline_from	query		string					false	"Срок не раньше (RFC 3339, включительно)"	Format(date-time)
// @Param			deadline_to		query		string					false	"Срок не позже (RFC 3339, включительно)"	Format(date-time)
// @Param			limit			query		int						false	"Размер страницы, от 1 до 100"				default(20)	minimum(1)	maximum(100)
// @Param			offset			query		int						false	"Смещение от начала списка"					default(0)	minimum(0)
// @Success		200				{object}	models.TaskListResponse	"Страница задач"
// @Failure		400				{object}	models.ErrorResponse	"Некорректные параметры фильтра или пагинации"
// @Failure		401				{object}	models.ErrorResponse	"Нет токена или токен недействителен"
// @Failure		500				{object}	models.ErrorResponse	"Внутренняя ошибка сервера"
// @Router			/api/v1/tasks [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseFilter(r.URL.Query())
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	page, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.resp.Error(w, r, err)

		return
	}

	h.resp.JSON(w, r, http.StatusOK, models.NewTaskListResponse(page))
}

func parseFilter(q url.Values) (models.TaskFilter, error) {
	var (
		filter models.TaskFilter
		err    error
	)

	if v := q.Get("status"); v != "" {
		status := models.Status(v)
		filter.Status = &status
	}

	if filter.AssigneeID, err = optionalUUID(q, "assignee_id"); err != nil {
		return models.TaskFilter{}, err
	}

	if filter.CreatorID, err = optionalUUID(q, "creator_id"); err != nil {
		return models.TaskFilter{}, err
	}

	if filter.DeadlineFrom, err = optionalTime(q, "deadline_from"); err != nil {
		return models.TaskFilter{}, err
	}

	if filter.DeadlineTo, err = optionalTime(q, "deadline_to"); err != nil {
		return models.TaskFilter{}, err
	}

	if filter.Limit, err = optionalInt(q, "limit"); err != nil {
		return models.TaskFilter{}, err
	}

	if filter.Offset, err = optionalInt(q, "offset"); err != nil {
		return models.TaskFilter{}, err
	}

	return filter, nil
}

func optionalUUID(q url.Values, key string) (*uuid.UUID, error) {
	v := q.Get(key)
	if v == "" {
		return nil, nil
	}

	id, err := uuid.Parse(v)
	if err != nil {
		return nil, &models.ValidationError{Field: key, Message: "must be a valid UUID"}
	}

	return &id, nil
}

func optionalTime(q url.Values, key string) (*time.Time, error) {
	v := q.Get(key)
	if v == "" {
		return nil, nil
	}

	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, &models.ValidationError{Field: key, Message: "must be an RFC 3339 timestamp"}
	}

	return &t, nil
}

func optionalInt(q url.Values, key string) (int, error) {
	v := q.Get(key)
	if v == "" {
		return 0, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, &models.ValidationError{Field: key, Message: "must be an integer"}
	}

	return n, nil
}
