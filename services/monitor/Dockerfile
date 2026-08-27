# syntax=docker/dockerfile:1

# Контекст сборки — корень репозитория, как и у остальных сервисов
# (сам monitor от platform не зависит, но команда сборки единообразна):
#
#   docker build -f services/monitor/Dockerfile -t banking/monitor .

FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOWORK=off

WORKDIR /src/services/monitor

# Сначала только манифесты — слой с зависимостями переживает правки кода.
COPY services/monitor/go.mod services/monitor/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY services/monitor/ ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/monitor .


FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/monitor /app/monitor

USER app

# Портов не слушает: monitor только опрашивает чужие health-проверки.
ENTRYPOINT ["/app/monitor"]
