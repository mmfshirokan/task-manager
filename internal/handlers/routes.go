package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpswagger "github.com/swaggo/http-swagger/v2"

	"github.com/mmfshirokan/task-manager/docs"
	"github.com/mmfshirokan/task-manager/internal/handlers/middleware"
	"github.com/mmfshirokan/task-manager/models"
)

// Routes собирает HTTP-роутер. Общее описание API ниже читает `make swagger`.
//
//	@title						Сервис управления задачами
//	@version					1.0
//	@description				REST API для управления задачами: регистрация и вход по JWT, создание, просмотр, изменение и удаление задач.
//	@description				Все авторизованные пользователи видят все задачи. Автор может менять любые поля и удалять задачу, исполнитель — только статус.
//	@description				Фоновый процесс переводит невыполненные задачи с истёкшим сроком в статус overdue.
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT-токен из ответа /api/v1/auth/login в формате: Bearer <токен>
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID, middleware.Logger(h.log), chimiddleware.Recoverer)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.resp.JSON(w, r, http.StatusNotFound, models.ErrorResponse{Error: "route not found"})
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		h.resp.JSON(w, r, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "method not allowed"})
	})

	r.Get("/healthz", h.healthz)
	r.Get("/swagger/*", httpswagger.Handler(httpswagger.InstanceName(docs.SwaggerInfo.InstanceName())))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", h.auth.Register)
		r.Post("/auth/login", h.auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(h.authSvc, h.resp))

			r.Post("/tasks", h.tasks.Create)
			r.Get("/tasks", h.tasks.List)
			r.Get("/tasks/{id}", h.tasks.Get)
			r.Patch("/tasks/{id}", h.tasks.Update)
			r.Delete("/tasks/{id}", h.tasks.Delete)
		})
	})

	return r
}

// @Summary		Проверка состояния
// @Description	Возвращает ok, если сервис запущен и принимает запросы.
// @Tags			Служебные
// @Produce		json
// @Success		200	{object}	models.HealthResponse	"Сервис работает"
// @Router			/healthz [get]
func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	h.resp.JSON(w, r, http.StatusOK, models.HealthResponse{Status: "ok"})
}
