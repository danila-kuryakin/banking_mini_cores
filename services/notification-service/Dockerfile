# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: сервису нужен соседний модуль platform.
#
#   docker build -f services/notification-service/Dockerfile -t banking/notification-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src/services/notification-service

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum /src/platform/
COPY services/notification-service/go.mod services/notification-service/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY platform/ /src/platform/
COPY services/notification-service/ ./

RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/notification-service ./cmd


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

# Сервис читает ./configs относительно рабочего каталога.
WORKDIR /app

COPY --from=builder /out/notification-service /app/notification-service
COPY services/notification-service/configs/config.yml /app/configs/config.yml

USER app

EXPOSE 50058

ENTRYPOINT ["/app/notification-service"]
