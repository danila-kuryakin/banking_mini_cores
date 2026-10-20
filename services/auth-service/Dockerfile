# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: сервису нужен соседний модуль platform.
#
#   docker build -f services/auth-service/Dockerfile -t banking/auth-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src/services/auth-service

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum /src/platform/
COPY services/auth-service/go.mod services/auth-service/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY platform/ /src/platform/
COPY services/auth-service/ ./

RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/auth-service ./cmd


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

# Сервис читает ./configs относительно рабочего каталога.
WORKDIR /app

COPY --from=builder /out/auth-service /app/auth-service
COPY services/auth-service/configs/config.yml /app/configs/config.yml

# Каталог ключей подписи — точка монтирования тома, писать в него должен app.
RUN mkdir -p /app/configs/jwt_keys && chown -R app:app /app/configs/jwt_keys

USER app

EXPOSE 50051 8081

ENTRYPOINT ["/app/auth-service"]
