# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: account-service подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/account-service/Dockerfile -t banking/account-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/account-service/go.mod services/account-service/go.sum ./services/account-service/

WORKDIR /src/services/account-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/account-service/ ./services/account-service/

WORKDIR /src/services/account-service
# Отдельного бинаря миграций нет: своих миграций у сервиса пока не заведено.
RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/account-service ./cmd/account-service


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/account-service /app/account-service
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/account-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50055

ENTRYPOINT ["/app/account-service"]
