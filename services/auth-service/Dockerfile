# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: auth-service подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/auth-service/Dockerfile -t banking/auth-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/auth-service/go.mod services/auth-service/go.sum ./services/auth-service/

WORKDIR /src/services/auth-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/auth-service/ ./services/auth-service/

WORKDIR /src/services/auth-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/auth-service ./cmd/auth-service && \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/auth-service /app/auth-service
COPY --from=builder /out/migrate /app/migrate
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/auth-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50051

ENTRYPOINT ["/app/auth-service"]
