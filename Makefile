.PHONY: run build generate test test-unit lint up down

run:
	go run .

build:
	go build -o bin/task-manager .

generate:
	go generate ./...

test:
	go test -race -count=1 ./...

test-unit:
	go test -short -race ./...

lint:
	golangci-lint run ./...

up:
	docker compose up --build -d

down:
	docker compose down
