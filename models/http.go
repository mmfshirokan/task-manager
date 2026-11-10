package models

import (
	"time"

	"github.com/google/uuid"
)

// Тела HTTP-запросов и ответов API. Комментарии к полям swag использует как
// описания моделей в Swagger-документации.

// CredentialsRequest — данные для регистрации и входа.
type CredentialsRequest struct {
	// Email пользователя
	Email string `json:"email" example:"alice@example.com"`
	// Пароль, от 8 до 72 байт
	Password string `json:"password" example:"secret123"`
}

// UserResponse — зарегистрированный пользователь.
type UserResponse struct {
	// Идентификатор пользователя
	ID uuid.UUID `json:"id" swaggertype:"string" format:"uuid" example:"f4eb7aa4-4a27-4bd2-a8e1-592e7a59f415"`
	// Email пользователя
	Email string `json:"email" example:"alice@example.com"`
	// Дата и время регистрации
	CreatedAt time.Time `json:"created_at" example:"2026-11-09T11:32:00Z"`
}

// TokenResponse — токен доступа, выданный при входе.
type TokenResponse struct {
	// JWT-токен доступа, передаётся в заголовке Authorization: Bearer <токен>
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// Тип токена, всегда Bearer
	TokenType string `json:"token_type" example:"Bearer"`
	// Время жизни токена в секундах
	ExpiresIn int64 `json:"expires_in" example:"86400"`
}

// CreateTaskRequest — данные новой задачи.
type CreateTaskRequest struct {
	// Название задачи, от 1 до 200 символов
	Title string `json:"title" example:"Подготовить релиз"`
	// Описание задачи, до 5000 символов
	Description string `json:"description" example:"Собрать список изменений для версии 1.0"`
	// Срок выполнения в формате RFC 3339, должен быть в будущем
	Deadline time.Time `json:"deadline" example:"2030-01-01T10:00:00Z"`
	// Исполнитель; если не указан, исполнителем становится автор задачи
	AssigneeID *uuid.UUID `json:"assignee_id,omitempty" swaggertype:"string" format:"uuid" example:"9fe291d3-2d67-41b0-8d1e-5928b9b3cca6"`
}

// UpdateTaskRequest — частичное обновление задачи: изменяются только переданные поля.
type UpdateTaskRequest struct {
	// Новое название, от 1 до 200 символов
	Title *string `json:"title,omitempty" example:"Подготовить релиз v2"`
	// Новое описание, до 5000 символов
	Description *string `json:"description,omitempty" example:"Добавить раздел о миграциях"`
	// Новый срок выполнения в формате RFC 3339
	Deadline *time.Time `json:"deadline,omitempty" example:"2030-02-01T10:00:00Z"`
	// Новый статус
	Status *Status `json:"status,omitempty" example:"in_progress"`
	// Новый исполнитель
	AssigneeID *uuid.UUID `json:"assignee_id,omitempty" swaggertype:"string" format:"uuid" example:"a09b1a4a-f390-4f19-a5c8-430ba98088a4"`
}

// TaskResponse — задача.
type TaskResponse struct {
	// Идентификатор задачи
	ID uuid.UUID `json:"id" swaggertype:"string" format:"uuid" example:"c908e324-5d57-4fb7-a2ba-7b36652e143d"`
	// Название задачи
	Title string `json:"title" example:"Подготовить релиз"`
	// Описание задачи
	Description string `json:"description" example:"Собрать список изменений для версии 1.0"`
	// Срок выполнения
	Deadline time.Time `json:"deadline" example:"2030-01-01T10:00:00Z"`
	// Статус: created — создана, in_progress — в работе, completed — выполнена, overdue — просрочена
	Status Status `json:"status" example:"created"`
	// Автор задачи
	CreatorID uuid.UUID `json:"creator_id" swaggertype:"string" format:"uuid" example:"f4eb7aa4-4a27-4bd2-a8e1-592e7a59f415"`
	// Исполнитель задачи
	AssigneeID uuid.UUID `json:"assignee_id" swaggertype:"string" format:"uuid" example:"9fe291d3-2d67-41b0-8d1e-5928b9b3cca6"`
	// Дата и время создания
	CreatedAt time.Time `json:"created_at" example:"2026-11-09T11:32:00Z"`
	// Дата и время последнего изменения
	UpdatedAt time.Time `json:"updated_at" example:"2026-11-09T11:40:00Z"`
}

// TaskListResponse — страница списка задач.
type TaskListResponse struct {
	// Задачи на странице, от новых к старым
	Items []TaskResponse `json:"items"`
	// Общее количество задач, подходящих под фильтр
	Total int `json:"total" example:"42"`
	// Размер страницы
	Limit int `json:"limit" example:"20"`
	// Смещение от начала списка
	Offset int `json:"offset" example:"0"`
}

// ErrorResponse — описание ошибки.
type ErrorResponse struct {
	// Текст ошибки
	Error string `json:"error" example:"title: must not be empty"`
}

// HealthResponse — состояние сервиса.
type HealthResponse struct {
	// Всегда ok, если сервис отвечает
	Status string `json:"status" example:"ok"`
}

func NewUserResponse(u User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}

func NewTokenResponse(t Token) TokenResponse {
	return TokenResponse{
		AccessToken: t.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(t.ExpiresIn.Seconds()),
	}
}

func NewTaskResponse(t Task) TaskResponse {
	return TaskResponse(t)
}

// NewTaskListResponse всегда возвращает ненулевой срез Items, чтобы пустая
// страница кодировалась в JSON как [], а не null.
func NewTaskListResponse(page TaskPage) TaskListResponse {
	items := make([]TaskResponse, 0, len(page.Items))
	for _, t := range page.Items {
		items = append(items, NewTaskResponse(t))
	}

	return TaskListResponse{Items: items, Total: page.Total, Limit: page.Limit, Offset: page.Offset}
}

func (req CreateTaskRequest) ToNewTask() NewTask {
	return NewTask(req)
}

func (req UpdateTaskRequest) ToTaskUpdate() TaskUpdate {
	return TaskUpdate(req)
}
