package rest

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler"
)

type RestServer struct {
	engine *gin.Engine
	router *Router
	server *http.Server
}

func NewRestServer(addr string, handler *handler.Handler, log *slog.Logger) *RestServer {
	engine := gin.New()
	engine.Use(gin.Recovery(), requestID(), logRequests(log))

	return &RestServer{
		engine: engine,
		router: NewRouter(handler),
		server: &http.Server{
			Addr:    addr,
			Handler: engine,
		},
	}
}

func (s *RestServer) Init() {
	s.router.Init(s.engine)
}

func (s *RestServer) Run() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *RestServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *RestServer) Addr() string {
	return s.server.Addr
}

func (s *RestServer) SwaggerURL() string {
	host, port, err := net.SplitHostPort(s.server.Addr)
	if err != nil {
		return "http://" + s.server.Addr + domain.SWAGGER_PAGE
	}

	switch host {
	case "", "0.0.0.0", "::":
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, port) + domain.SWAGGER_PAGE
}
