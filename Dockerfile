FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/task-manager .

FROM alpine:3

RUN adduser -D -u 10001 app
USER app

COPY --from=build /out/task-manager /usr/local/bin/task-manager

EXPOSE 8080
ENTRYPOINT ["task-manager"]
