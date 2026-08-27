// Package httpapi is the inbound HTTP adapter of api-gateway: grpc-gateway
// переводит REST-запросы в gRPC-вызовы сервисов по правилам google.api.http
// из proto.
//
// Своей бизнес-логики здесь нет - только транспорт, корреляция запросов и
// жизненный цикл серверов.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	gwconfig "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/account/v1"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/antifraud/v1"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/auth/v1"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
	documentv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/document/v1"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/kyc/v1"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/ledger/v1"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/swaggerui"
)

// Gateway — это HTTP-сервер и соединения gRPC за ним.
type Gateway struct {
	server *http.Server
	conns  []*grpc.ClientConn
}

const specPath = "/openapi.json"

func New(cfg *gwconfig.Config, log *slog.Logger) error {
	gw := &Gateway{}

	// NotifyContext сам снимает обработчик сигнала по stop() - в отличие от
	// голого signal.Notify в горутине, которая висела бы вечно, если серверы
	// упадут, так и не дождавшись сигнала.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Соединения с gRPC-бэкендами. NewClient не ходит в сеть сразу -
	// подключение установится лениво, при первом вызове.
	defer gw.closeConns()

	authConn, err := gw.dial(cfg.Upstreams.Auth)
	if err != nil {
		return err
	}

	customerConn, err := gw.dial(cfg.Upstreams.Customer)
	if err != nil {
		return err
	}

	accountConn, err := gw.dial(cfg.Upstreams.Account)
	if err != nil {
		return err
	}

	antifraudConn, err := gw.dial(cfg.Upstreams.Antifraud)
	if err != nil {
		return err
	}

	kycConn, err := gw.dial(cfg.Upstreams.KYC)
	if err != nil {
		return err
	}

	documentConn, err := gw.dial(cfg.Upstreams.Document)
	if err != nil {
		return err
	}

	ledgerConn, err := gw.dial(cfg.Upstreams.Ledger)
	if err != nil {
		return err
	}

	// gwMux - сгенерированный grpc-gateway мост: разбирает HTTP-запрос по
	// правилам google.api.http из proto и вызывает соответствующий gRPC-метод.
	gwMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   protojson.MarshalOptions{EmitUnpopulated: true},
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
		runtime.WithMetadata(gatewayMetadata),
	)

	if err := authv1.RegisterAuthServiceHandler(ctx, gwMux, authConn); err != nil {
		return err
	}
	if err := customerv1.RegisterCustomerServiceHandler(ctx, gwMux, customerConn); err != nil {
		return err
	}
	if err := accountv1.RegisterAccountServiceHandler(ctx, gwMux, accountConn); err != nil {
		return err
	}
	if err := antifraudv1.RegisterAntifraudServiceHandler(ctx, gwMux, antifraudConn); err != nil {
		return err
	}
	if err := kycv1.RegisterKycServiceHandler(ctx, gwMux, kycConn); err != nil {
		return err
	}
	if err := documentv1.RegisterDocumentServiceHandler(ctx, gwMux, documentConn); err != nil {
		return err
	}
	if err := ledgerv1.RegisterLedgerServiceHandler(ctx, gwMux, ledgerConn); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/", gwMux)
	mux.Handle(specPath, swaggerui.SpecHandler())
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", swaggerui.Handler(specPath)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/swagger/", http.StatusFound)
	})

	httpAddr := cfg.RestServer.GetAddr()
	gw.server = &http.Server{
		Addr: httpAddr,
		// requestID снаружи логов: строка лога должна уже содержать id.
		Handler:           requestID(logRequests(log, mux)),
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
	grpcSrv := grpc.NewServer()
	healthSrv := grpc_server.HealthServer(grpcSrv)
	reflection.Register(grpcSrv)

	healthAddr := cfg.GRPCServer.GetAddr()
	healthLis, err := net.Listen("tcp", healthAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", healthAddr, err)
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
			"addr", httpAddr,
			"health", healthAddr,
			"auth", cfg.Upstreams.Auth,
			"customer", cfg.Upstreams.Customer,
			"swagger", "http://"+httpAddr+"/swagger/",
		)
		if err := gw.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down", "timeout", cfg.ShutdownTimeout)
		// Сначала объявляем себя NOT_SERVING, чтобы monitor увидел штатное
		// выключение, и только потом закрываем серверы.
		healthSrv.Shutdown()
		grpcSrv.GracefulStop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		return gw.server.Shutdown(shutdownCtx)
	}
}

// dial открывает соединение с апстримом и запоминает его, чтобы closeConns
// закрыл всё разом.
//
// StatsHandler тот же, что ставит себе platform/grpc_server на входящих
// вызовах, - так спан клиента и спан сервера склеиваются в одну трассу.
func (g *Gateway) dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	g.conns = append(g.conns, conn)

	return conn, nil
}

func (g *Gateway) closeConns() {
	for _, conn := range g.conns {
		_ = conn.Close()
	}
}
