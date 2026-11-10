.PHONY: run build generate swagger test test-unit lint up down

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

generate: ## Кодогенерация
	go generate ./...

swagger: ## Обновление Swagger-документации (docs/) по аннотациям хендлеров
	go tool swag fmt -d ./internal/handlers
	go tool swag init -g routes.go -d ./internal/handlers,./models -o ./docs --outputTypes go,json,yaml

test: ## Юнит и интеграционные тесты
	gotestsum --format=testname -- -race ./...

lint: ## Запуск линта
	golangci-lint run ./...

up: ## Запуск приложения в докере
	docker compose up --build -d

down: ## Завершение приложения в докере
	docker compose down
