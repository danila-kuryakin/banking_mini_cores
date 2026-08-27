# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: api-gateway подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/api-gateway/Dockerfile -t banking/api-gateway .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./services/api-gateway/

WORKDIR /src/services/api-gateway
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/api-gateway/ ./services/api-gateway/

WORKDIR /src/services/api-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./cmd/api-gateway


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/api-gateway /app/api-gateway
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/api-gateway/configs/ /app/configs/

USER app

# HTTP (REST + Swagger UI) и gRPC-порт health-проверки.
EXPOSE 8080 50059

ENTRYPOINT ["/app/api-gateway"]
