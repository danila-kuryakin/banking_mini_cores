# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: ledger-service подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/ledger-service/Dockerfile -t banking/ledger-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/ledger-service/go.mod services/ledger-service/go.sum ./services/ledger-service/

WORKDIR /src/services/ledger-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/ledger-service/ ./services/ledger-service/

WORKDIR /src/services/ledger-service
# Отдельного бинаря миграций нет: своих миграций у сервиса пока не заведено.
RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/ledger-service ./cmd/ledger-service


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/ledger-service /app/ledger-service
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/ledger-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50056

ENTRYPOINT ["/app/ledger-service"]
