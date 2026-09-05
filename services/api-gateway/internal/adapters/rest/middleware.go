package rest

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	"github.com/danila-kuryakin/banking_mini_cores/platform/interceptors"
)

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(interceptors.RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
			c.Request.Header.Set(interceptors.RequestIDHeader, id)
		}

		c.Header(interceptors.RequestIDHeader, id)

		ctx := metadata.AppendToOutgoingContext(c.Request.Context(), interceptors.RequestIDHeader, id)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func logRequests(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"took", time.Since(start),
			"request_id", c.Request.Header.Get(interceptors.RequestIDHeader),
		}

		if errs := c.Errors.Errors(); len(errs) > 0 {
			attrs = append(attrs, "errors", errs)
		}

		log.Log(c.Request.Context(), levelFor(status), "http", attrs...)
	}
}

func levelFor(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
