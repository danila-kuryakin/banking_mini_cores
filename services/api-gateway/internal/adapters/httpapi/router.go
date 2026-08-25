package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/auth/v1"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/swaggerui"

	gwconfig "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
)

// Gateway — это HTTP-сервер и соединения gRPC за ним.
type Gateway struct {
	server *http.Server
	conns  []*grpc.ClientConn
}

const specPath = "/openapi.json"

func New(cfg gwconfig.Config, log *slog.Logger) error {
	gw := &Gateway{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Соединения с gRPC-бэкендами. NewClient не ходит в сеть сразу -
	// подключение установится лениво, при первом вызове.
	authConn, err := gw.dial(cfg.Upstreams.Auth)
	if err != nil {
		return err
	}
	defer func() { _ = authConn.Close() }()

	customerConn, err := gw.dial(cfg.Upstreams.Customer)
	if err != nil {
		return err
	}
	defer func() { _ = customerConn.Close() }()

	// gwMux - сгенерированный grpc-gateway мост: разбирает HTTP-запрос по
	// правилам google.api.http из proto и вызывает соответствующий gRPC-метод.
	gwMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   protojson.MarshalOptions{EmitUnpopulated: true},
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
	)

	if err := authv1.RegisterAuthServiceHandler(ctx, gwMux, authConn); err != nil {
		return err
	}
	if err := customerv1.RegisterCustomerServiceHandler(ctx, gwMux, customerConn); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/", gwMux)
	mux.Handle(specPath, swaggerui.SpecHandler())
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", swaggerui.Handler(specPath)))
	//mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
	//	w.WriteHeader(http.StatusOK)
	//	_, _ = w.Write([]byte("ok"))
	//})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/swagger/", http.StatusFound)
	})

	gw.server = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           logRequests(log, mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.RequestTimeout,
		WriteTimeout:      cfg.RequestTimeout + 5*time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // ограничение размера заголовка
	}

	errCh := make(chan error, 2)
	// Сам gateway отвечает по HTTP, но health-проверку отдаёт по gRPC - тем же
	// протоколом grpc.health.v1.Health, что и остальные сервисы, чтобы monitor
	// не пришлось учить двум способам опроса.
	healthSrv := health.NewServer()
	grpcSrv := grpc.NewServer()
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	reflection.Register(grpcSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	healthLis, err := net.Listen("tcp", cfg.GatewayAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.GatewayAddr, err)
	}
	go func() {
		// Serve возвращает ErrServerStopped после GracefulStop - это штатный
		// выход, а не сбой, поэтому в errCh он не попадает.
		if err := grpcSrv.Serve(healthLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- err
		}
	}()

	go func() {
		log.Info("http listening",
			"addr", cfg.HTTPAddr,
			"health", cfg.GatewayAddr,
			"auth", cfg.Upstreams.Auth,
			//"customer", cfg.Upstreams.Customer,
			"swagger", "http://localhost"+cfg.HTTPAddr+"/swagger/",
		)
		if err := gw.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-stop:
		log.Info("shutting down")
		// Сначала объявляем себя NOT_SERVING, чтобы monitor увидел штатное
		// выключение, и только потом закрываем серверы.
		healthSrv.Shutdown()
		grpcSrv.GracefulStop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		return gw.server.Shutdown(shutdownCtx)
	}
}

func (g *Gateway) dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http", "method", r.Method, "path", r.URL.Path, "took", time.Since(start))
	})
}
