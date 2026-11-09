package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/mmfshirokan/task-manager/models"
)

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTaskFilter(r.URL.Query())
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	page, err := h.tasks.ListTasks(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)

		return
	}

	items := make([]taskResponse, 0, len(page.Items))
	for _, t := range page.Items {
		items = append(items, toTaskResponse(t))
	}

	h.writeJSON(w, r, http.StatusOK, taskListResponse{
		Items:  items,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	})
}

// parseTaskFilter reads status, assignee_id, deadline_from, deadline_to
// (RFC 3339), limit and offset from the query string.
func parseTaskFilter(q url.Values) (models.TaskFilter, error) {
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
