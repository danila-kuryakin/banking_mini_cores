package grpc_server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	// defaultShutdownTimeout - сколько ждём GracefulStop, прежде чем рвать
	// соединения принудительно. Без потолка зависший вызов держит процесс вечно.
	defaultShutdownTimeout = 15 * time.Second

	// defaultHandlerTimeout - верхняя граница жизни одного unary-хендлера.
	// Намеренно щедрая: это предохранитель от зависаний, а не SLA. Сервис,
	// которому нужно строже, задаёт своё через WithHandlerTimeout.
	defaultHandlerTimeout = 15 * time.Second
)

// options - всё, что настраивается через Option. Нулевое значение поля
// означает "как по умолчанию", поэтому NewServer(addr, logger, WithServices(...))
// без единой дополнительной опции остаётся полноценным вызовом.
type options struct {
	services []RegisterFunc

	unary  []grpc.UnaryServerInterceptor
	stream []grpc.StreamServerInterceptor

	grpcOpts []grpc.ServerOption

	shutdownTimeout time.Duration
	handlerTimeout  time.Duration
	metricsReg      prometheus.Registerer

	disableReflection   bool
	disableTracing      bool
	disableMetrics      bool
	disableInterceptors bool
}

func newOptions(opts ...Option) options {
	o := options{
		shutdownTimeout: defaultShutdownTimeout,
		handlerTimeout:  defaultHandlerTimeout,
		metricsReg:      prometheus.DefaultRegisterer,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

type Option func(*options)

// WithServices добавляет регистрации сервисов. Вызывается сколько угодно раз и
// принимает сколько угодно RegisterFunc - оба варианта эквивалентны, выбирайте
// что читаемее на месте вызова.
func WithServices(fns ...RegisterFunc) Option {
	return func(o *options) { o.services = append(o.services, fns...) }
}

// WithUnaryInterceptors добавляет интерцепторы сервиса. Они встают ПОСЛЕ
// базовой цепочки платформы, то есть ближе к хендлеру: аутентификация,
// валидация и маппинг доменных ошибок уже защищены Recovery и попадают в
// логи и метрики.
//
// Порядок внутри вызова сохраняется: первый переданный - самый внешний.
func WithUnaryInterceptors(i ...grpc.UnaryServerInterceptor) Option {
	return func(o *options) { o.unary = append(o.unary, i...) }
}

// WithStreamInterceptors - то же для стримов.
func WithStreamInterceptors(i ...grpc.StreamServerInterceptor) Option {
	return func(o *options) { o.stream = append(o.stream, i...) }
}

// WithServerOptions - лазейка для сырых grpc.ServerOption: TLS-креды,
// keepalive-политики, лимиты на размер сообщения.
func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(o *options) { o.grpcOpts = append(o.grpcOpts, opts...) }
}

// WithHandlerTimeout меняет дедлайн unary-хендлера. Нулевое значение
// выключает интерцептор Timeout совсем.
func WithHandlerTimeout(d time.Duration) Option {
	return func(o *options) { o.handlerTimeout = d }
}

func WithShutdownTimeout(d time.Duration) Option {
	return func(o *options) { o.shutdownTimeout = d }
}

// WithMetricsRegisterer подменяет реестр Prometheus - нужно в тестах, чтобы
// не тащить коллекторы в глобальный DefaultRegisterer.
func WithMetricsRegisterer(reg prometheus.Registerer) Option {
	return func(o *options) { o.metricsReg = reg }
}

// WithoutReflection выключает server reflection. В проде её обычно гасят,
// чтобы не отдавать схему API наружу; локально она нужна для grpcurl.
func WithoutReflection() Option {
	return func(o *options) { o.disableReflection = true }
}

// WithoutTracing убирает otel stats handler - для сервисов, у которых нет
// сконфигурированного экспортёра.
func WithoutTracing() Option {
	return func(o *options) { o.disableTracing = true }
}

func WithoutMetrics() Option {
	return func(o *options) { o.disableMetrics = true }
}

// WithoutDefaultInterceptors отключает всю базовую цепочку платформы.
// Аварийный выход для нестандартного сервиса; в обычной ситуации не нужен.
func WithoutDefaultInterceptors() Option {
	return func(o *options) { o.disableInterceptors = true }
}
