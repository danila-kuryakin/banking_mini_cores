package interceptors

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var testInfo = &grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/Login"}

const testTimeout = 50 * time.Millisecond

// call прогоняет handler через ту же цепочку, что собирает grpc,
// в том же порядке. Логи глушим, метрики уводим в локальный реестр.
func call(ctx context.Context, handler grpc.UnaryHandler) (any, error) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	chain := []grpc.UnaryServerInterceptor{
		Recovery(log),
		RequestID(),
		Metrics(prometheus.NewRegistry()),
		Logging(log),
		Timeout(testTimeout),
	}

	next := handler
	for i := len(chain) - 1; i >= 0; i-- {
		ic, inner := chain[i], next
		next = func(c context.Context, r any) (any, error) { return ic(c, r, testInfo, inner) }
	}
	return next(ctx, nil)
}

func TestPanicBecomesInternal(t *testing.T) {
	_, err := call(context.Background(), func(context.Context, any) (any, error) {
		panic("boom")
	})
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("code = %v, want Internal", got)
	}
}

func TestRequestIDTakenFromMetadata(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs(RequestIDHeader, "abc-123"))

	var got string
	if _, err := call(ctx, func(c context.Context, _ any) (any, error) {
		got = RequestIDFrom(c)
		return "ok", nil
	}); err != nil {
		t.Fatal(err)
	}
	if got != "abc-123" {
		t.Fatalf("request id = %q, want abc-123", got)
	}
}

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	var got string
	if _, err := call(context.Background(), func(c context.Context, _ any) (any, error) {
		got = RequestIDFrom(c)
		return "ok", nil
	}); err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("request id not generated")
	}
}

// Голый context.DeadlineExceeded из хендлера уходил бы клиенту как Unknown -
// Timeout обязан перевести его в осмысленный код.
func TestTimeoutReportsDeadlineExceeded(t *testing.T) {
	_, err := call(context.Background(), func(c context.Context, _ any) (any, error) {
		<-c.Done()
		return nil, c.Err()
	})
	if got := status.Code(err); got != codes.DeadlineExceeded {
		t.Fatalf("code = %v, want DeadlineExceeded", got)
	}
}

// Дедлайн клиента жёстче нашего - интерцептор не должен его ослаблять.
func TestClientDeadlineIsNotLoosened(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _ = call(ctx, func(c context.Context, _ any) (any, error) {
		<-c.Done()
		return nil, c.Err()
	})
	if elapsed := time.Since(start); elapsed >= testTimeout {
		t.Fatalf("waited %v, client deadline was 5ms", elapsed)
	}
}

// Статус, который хендлер вернул осознанно, Timeout перетирать не должен.
func TestHandlerStatusIsPreserved(t *testing.T) {
	want := status.Error(codes.NotFound, "user not found")
	_, err := call(context.Background(), func(context.Context, any) (any, error) {
		return nil, want
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("code = %v, want NotFound", got)
	}
}

func TestSplitFullMethod(t *testing.T) {
	tests := map[string]struct{ service, method string }{
		"/auth.v1.AuthService/Login":   {"auth.v1.AuthService", "Login"},
		"/grpc.health.v1.Health/Check": {"grpc.health.v1.Health", "Check"},
		"malformed":                    {"unknown", "malformed"},
	}
	for in, want := range tests {
		service, method := splitFullMethod(in)
		if service != want.service || method != want.method {
			t.Errorf("splitFullMethod(%q) = %q, %q; want %q, %q",
				in, service, method, want.service, want.method)
		}
	}
}

// Второй NewServer в одном процессе (типичный случай в тестах) не должен
// падать на AlreadyRegisteredError.
func TestMetricsSurvivesReRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	Metrics(reg)
	Metrics(reg)
}
