# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: notification-service подтягивает
# platform через replace ../../platform, поэтому одного каталога сервиса
# не хватает.
#
#   docker build -f services/notification-service/Dockerfile -t banking/notification-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/notification-service/go.mod services/notification-service/go.sum ./services/notification-service/

WORKDIR /src/services/notification-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/notification-service/ ./services/notification-service/

WORKDIR /src/services/notification-service
# Миграций тут нет: сервис держит уведомления в памяти, база ему не нужна.
RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/notification-service ./cmd/notification-service


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/notification-service /app/notification-service
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/notification-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50058

ENTRYPOINT ["/app/notification-service"]
