# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: customer-service подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/customer-service/Dockerfile -t banking/customer-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/customer-service/go.mod services/customer-service/go.sum ./services/customer-service/

WORKDIR /src/services/customer-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/customer-service/ ./services/customer-service/

WORKDIR /src/services/customer-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/customer-service ./cmd/customer-service && \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/customer-service /app/customer-service
COPY --from=builder /out/migrate /app/migrate
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/customer-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50052

ENTRYPOINT ["/app/customer-service"]
