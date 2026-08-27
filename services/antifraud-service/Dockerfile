# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: antifraud-service подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/antifraud-service/Dockerfile -t banking/antifraud-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/antifraud-service/go.mod services/antifraud-service/go.sum ./services/antifraud-service/

WORKDIR /src/services/antifraud-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/antifraud-service/ ./services/antifraud-service/

WORKDIR /src/services/antifraud-service
# Отдельного бинаря миграций нет: своих миграций у сервиса пока не заведено.
RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/antifraud-service ./cmd/antifraud-service


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/antifraud-service /app/antifraud-service
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/antifraud-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50057

ENTRYPOINT ["/app/antifraud-service"]
