# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория: kyc-service подтягивает platform
# через replace ../../platform, поэтому одного каталога сервиса не хватает.
#
#   docker build -f services/kyc-service/Dockerfile -t banking/kyc-service .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0     GOOS=linux     GOWORK=off

WORKDIR /src

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY platform/go.mod platform/go.sum ./platform/
COPY services/kyc-service/go.mod services/kyc-service/go.sum ./services/kyc-service/

WORKDIR /src/services/kyc-service
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /src
COPY platform/ ./platform/
COPY services/kyc-service/ ./services/kyc-service/

WORKDIR /src/services/kyc-service
# Отдельного бинаря миграций нет: своих миграций у сервиса пока не заведено.
RUN --mount=type=cache,target=/go/pkg/mod     --mount=type=cache,target=/root/.cache/go-build     go build -trimpath -ldflags="-s -w" -o /out/kyc-service ./cmd/kyc-service


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata &&     adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/kyc-service /app/kyc-service
# config.Read ищет ./configs относительно рабочего каталога.
COPY services/kyc-service/configs/ /app/configs/

USER app

# gRPC API сервиса.
EXPOSE 50053

ENTRYPOINT ["/app/kyc-service"]
