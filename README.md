# Task Manager

REST API для управления задачами: регистрация и вход по JWT (bearer-токен), хранение
в Postgres и фоновый процесс, который переводит просроченные задачи в статус `overdue`.

## Содержание
- [Требования](#требования)
- [Запуск сервиса](#запуск-сервиса)
  - [Docker Compose](#docker-compose)
  - [Локальный запуск](#локальный-запуск)
- [Env](#env)
- [Команды Makefile](#команды-makefile)
- [Тесты и моки](#тесты-и-моки)
- [Swagger](#swagger)
- [Правила доступа](#правила-доступа)
- [API](#api)
- [Структура проекта](#структура-проекта)
- [Схема зависимостей компонентов](#схема-зависимостей-компонентов)

## Требования

- Go 1.27+
- Docker с Docker Compose — для запуска стека и интеграционных тестов
- [gotestsum](https://github.com/gotestyourself/gotestsum) — для `make test`:
  `go install gotest.tools/gotestsum@latest`
- [golangci-lint](https://golangci-lint.run/) v2 — для `make lint`:
  `brew install golangci-lint` или официальный скрипт:
  `curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin`

`mockgen` и `swag` устанавливать отдельно не нужно — они подключены как Go tools в `go.mod`.

## Запуск сервиса

### Docker Compose

```sh
make up      # docker compose up --build -d
```

Compose поднимает Postgres, дожидается его healthcheck и запускает сервис. При старте
сервис сам применяет миграции и слушает <http://localhost:8080>.

Файл `.env` в корне проекта необязателен: у всех переменных есть значения по умолчанию
(см. [Env](#env)). Если порт `5432` или `8080` занят, переопределите его в `.env` или
прямо в команде: `POSTGRES_PORT=55432 HTTP_PORT=8081 make up`.

Остановка: `make down` (или `docker compose down -v`, чтобы также удалить том с базой).

Проверка:

```sh
curl -s http://localhost:8080/healthz
# -> {"status":"ok"}
```

### Локальный запуск

1. Поднять Postgres (из Compose или свой):
   ```sh
   docker compose up -d db
   ```
2. Создать файл `.env` в корне проекта и указать как минимум `DATABASE_URL` и `JWT_SECRET`
   (пример — в разделе [Env](#env)). Вместо файла можно экспортировать переменные окружения.
3. Запустить сервис:
   ```sh
   go run .
   ```

Сервис читает `.env` из текущей директории, если файл существует; реальные переменные
окружения имеют приоритет над значениями из `.env`.

## Env

```text
# Сервис
HTTP_PORT=8080                                                             # Порт HTTP-сервера. Optional, default 8080.
DATABASE_URL=postgres://postgres:postgres@localhost:5432/tasks             # Строка подключения к Postgres. Required.
JWT_SECRET=change-me-to-a-long-random-string                               # HMAC-секрет для подписи токенов. Required.
JWT_TTL=24h                                                                # Время жизни access-токена. Optional, default 24h.
OVERDUE_INTERVAL=1m                                                        # Период запуска фоновой задачи по просроченным задачам. Optional, default 1m.
SHUTDOWN_TIMEOUT=10s                                                       # Таймаут graceful shutdown для активных запросов. Optional, default 10s.
LOG_LEVEL=info                                                             # Уровень логов: debug, info, warn, error. Optional, default info.

# Контейнер Postgres (только для docker compose)
POSTGRES_USER=postgres                                                     # Optional, default postgres.
POSTGRES_PASSWORD=postgres                                                 # Optional, default postgres.
POSTGRES_DB=tasks                                                          # Optional, default tasks.
POSTGRES_PORT=5432                                                         # Порт Postgres на хосте. Optional, default 5432.
```

В Docker Compose `DATABASE_URL` и `JWT_SECRET` для контейнера сервиса собираются
автоматически из `POSTGRES_*` и `JWT_SECRET`, а `HTTP_PORT` задаёт порт на хосте
(внутри контейнера сервис всегда слушает `8080`).

## Команды Makefile

| Команда         | Описание                                                               |
|-----------------|------------------------------------------------------------------------|
| `make help`     | Список доступных целей                                                 |
| `make up`       | `docker compose up --build -d` — запуск стека в Docker                 |
| `make down`     | `docker compose down` — остановка стека                                |
| `make test`     | Все тесты с `-race`, включая интеграционные (нужен Docker)             |
| `make lint`     | `golangci-lint run ./...` с конфигом из `.golangci.yml`                |
| `make generate` | Перегенерация gomock-моков (`go generate ./...`)                       |
| `make swagger`  | Перегенерация Swagger-документации в `docs/` по аннотациям хендлеров   |

Только юнит-тесты, без Docker: `go test -short ./...`.

## Тесты и моки

Интеграционные тесты находятся в `internal/repository/repository_test.go`. Они поднимают
временный контейнер `postgres:17-alpine` через
[dockertest](https://github.com/ory/dockertest), применяют миграции и проверяют
репозитории `users` и `tasks`. Build-тега у них нет, они пропускаются флагом `-short`.

Моки генерируются с помощью [mockgen](https://github.com/uber-go/mock), который подключён через `go.mod`. Каждый потребитель объявляет нужные ему интерфейсы в главном файле своего
пакета (`handlers.go`, `auth.go`, `tasks.go`, `cron.go`), там же стоит директива
`//go:generate`; моки складываются в директорию `mocks/` пакета.

## Swagger

Интерактивная документация доступна у запущенного сервиса по адресу
<http://localhost:8080/swagger/index.html>, сырая спецификация — `/swagger/doc.json`.
Для защищённых эндпоинтов нажмите **Authorize** и введите `Bearer <token>`, где `token`
получен из `/api/v1/auth/login`.

Спецификация генерируется [swaggo/swag](https://github.com/swaggo/swag) из аннотаций на
методах хендлеров в `internal/handlers/` и из комментариев к полям DTO в `models/http.go`.
После изменения эндпоинта или DTO выполните `make swagger` и закоммитьте обновлённые
файлы в `docs/` (`docs.go`, `swagger.json`, `swagger.yaml`).

## Правила доступа

- Любой авторизованный пользователь видит список и детали всех задач.
- **Автор** задачи может менять любые её поля и единственный, кто может её удалить.
- **Исполнитель** может менять только поле `status`.
- Все остальные получают `403 Forbidden` при изменении или удалении.

Статусы задачи: `created`, `in_progress`, `completed`, `overdue`. Фоновый процесс
переводит задачи в статусах `created` и `in_progress` с истёкшим `deadline` в `overdue`.

## API

Базовый путь: `/api/v1`. Все тела запросов и ответов — JSON; ошибки имеют вид
`{"error": "message"}`.

| Метод    | Путь              | Auth   | Описание                                  |
|----------|-------------------|--------|-------------------------------------------|
| `POST`   | `/auth/register`  | —      | Регистрация пользователя                  |
| `POST`   | `/auth/login`     | —      | Получение bearer-токена                   |
| `POST`   | `/tasks`          | Bearer | Создание задачи                           |
| `GET`    | `/tasks`          | Bearer | Список задач с фильтрами и пагинацией     |
| `GET`    | `/tasks/{id}`     | Bearer | Получение задачи                          |
| `PATCH`  | `/tasks/{id}`     | Bearer | Частичное обновление задачи               |
| `DELETE` | `/tasks/{id}`     | Bearer | Удаление задачи                           |
| `GET`    | `/healthz`        | —      | Liveness-проверка (вне `/api/v1`)         |

Коды ответов: `400` — невалидный ввод, `401` — нет/невалидный токен или неверные
учётные данные, `403` — действие запрещено, `404` — не найдено, `409` — email уже
зарегистрирован, `500` — внутренняя ошибка.

Валидация: корректный email, пароль 8–72 байта, `title` 1–200 символов, `description`
до 5000 символов, `deadline` новой задачи — RFC 3339 и в будущем.

Фильтры списка: `status`, `assignee_id`, `deadline_from` / `deadline_to` (RFC 3339),
`limit` (1–100, по умолчанию 20), `offset`.

## Структура проекта

```text
task-manager/
├── app/                      # Сборка зависимостей: подключение к БД, HTTP-сервер, фоновая задача, graceful shutdown
├── config/                   # Загрузка конфигурации (.env + переменные окружения)
├── docs/                     # Сгенерированная Swagger-спецификация (make swagger)
├── models/                   # Доменные типы, ошибки, тела HTTP-запросов и ответов (http.go)
├── schema/
│   ├── migrations/           # SQL-миграции (goose)
│   ├── source.go             # Встроенные миграции (embed)
│   └── migrate.go            # Migrate — выполняется при старте и в интеграционных тестах
├── internal/                 # Внутренние пакеты, недоступны для импорта извне
│   ├── repository/           # Слой доступа к данным (Postgres через pgx)
│   │   ├── repository.go     # New(db) -> Users, Tasks
│   │   ├── users/            # Таблица users: Create, GetByEmail, GetByID
│   │   └── tasks/            # Таблица tasks: Create, Get, List, Update, Delete, MarkOverdue
│   ├── service/              # Бизнес-логика
│   │   ├── service.go        # New(...) -> Auth, Tasks
│   │   ├── auth/             # Register, Login, ParseToken (bcrypt, JWT)
│   │   └── tasks/            # Create, Get, List, Update, Delete, MarkOverdue (валидация, права)
│   ├── handlers/             # HTTP-слой (chi)
│   │   ├── handlers.go       # Интерфейсы сервисов, Handler
│   │   ├── routes.go         # Роутер и таблица маршрутов
│   │   ├── endpoints/auth/   # register, login
│   │   ├── endpoints/tasks/  # create, get, list, update, delete
│   │   ├── middleware/       # Bearer-аутентификация, логирование запросов
│   │   └── response/         # JSON-ответы, маппинг ошибок в статусы, декодирование тела
│   └── cron/                 # Фоновая задача, помечающая просроченные задачи
├── docker-compose.yml        # Postgres + сервис
├── Dockerfile                # Multi-stage сборка сервиса
├── Makefile                  # Команды разработки
└── main.go                   # Входная точка запуска сервиса, вызывает app.Run
```

В каждом пакете есть главный файл с именем пакета (структура, `New`, используемые
интерфейсы) и по одному файлу на операцию. Слои общаются через интерфейсы, объявленные
на стороне потребителя: handlers / cron → service → repository.

## Схема зависимостей компонентов

Направление стрелок показывает зависимость компонентов, так `a → b` значит, что
компонент `a` зависит от компонента `b`. Подписи на стрелках — интерфейсы, через
которые потребитель обращается к следующему слою:

```mermaid
flowchart TD
    mainNode["main.go"] --> handlerLayer["internal/handler (HTTP)"]
    mainNode --> cronLayer["internal/cron (OverdueJob)"]
    handlerLayer -->|"TaskService / AuthService interfaces"| serviceLayer["internal/service"]
    cronLayer -->|"OverdueMarker interface"| serviceLayer
    serviceLayer -->|"TaskRepository / UserRepository interfaces"| repoLayer["internal/repository"]
    repoLayer --> pg[(Postgres)]
```
