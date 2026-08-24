// Package checker опрашивает health-проверки gRPC-сервисов и печатает результат.
package checker

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// down - состояние, которого нет в протоколе: сервис не ответил вовсе.
// SERVING/NOT_SERVING приходят от самого сервиса, DOWN придумывает monitor.
const down = "DOWN"

// Target - сервис, за которым следим.
type Target struct {
	Name string
	Addr string
}

// Checker держит по одному gRPC-соединению на цель и опрашивает их health.
type Checker struct {
	log     *slog.Logger
	targets []Target
	clients map[string]healthpb.HealthClient
	conns   []*grpc.ClientConn

	// last - статус с прошлого опроса, чтобы отдельно логировать изменения.
	last map[string]string
}

// New создаёт Checker и соединения до всех целей. Соединения ленивые: если
// сервис ещё не поднят, ошибки здесь не будет - она появится при опросе.
func New(log *slog.Logger, targets []Target) (*Checker, error) {
	c := &Checker{
		log:     log,
		targets: targets,
		clients: make(map[string]healthpb.HealthClient, len(targets)),
		last:    make(map[string]string, len(targets)),
	}

	for _, t := range targets {
		conn, err := grpc.NewClient(t.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("подключение к %s (%s): %w", t.Name, t.Addr, err)
		}
		c.conns = append(c.conns, conn)
		c.clients[t.Name] = healthpb.NewHealthClient(conn)
	}

	return c, nil
}

// Close закрывает все соединения.
func (c *Checker) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}

// CheckAll опрашивает все цели параллельно и пишет результат в лог: одну
// сводную строку на весь круг и отдельную строку на каждое изменение статуса.
func (c *Checker) CheckAll(ctx context.Context) {
	var (
		mu       sync.Mutex
		statuses = make(map[string]string, len(c.targets))
		errs     = make(map[string]error, len(c.targets))
		wg       sync.WaitGroup
	)

	for _, t := range c.targets {
		wg.Add(1)
		go func(t Target) {
			defer wg.Done()

			status, err := c.check(ctx, t)

			mu.Lock()
			defer mu.Unlock()
			statuses[t.Name] = status
			if err != nil {
				errs[t.Name] = err
			}
		}(t)
	}
	wg.Wait()

	attrs := make([]any, 0, len(c.targets)*2)
	for _, t := range c.targets {
		attrs = append(attrs, t.Name, statuses[t.Name])
	}
	c.log.Info("health", attrs...)

	c.reportChanges(statuses, errs)
}

// check возвращает статус одной цели: SERVING, NOT_SERVING или DOWN с ошибкой.
func (c *Checker) check(ctx context.Context, t Target) (string, error) {
	resp, err := c.clients[t.Name].Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return down, err
	}
	return resp.GetStatus().String(), nil
}

// reportChanges логирует только переходы между статусами - по ним видно момент
// падения или восстановления, не вчитываясь в поток одинаковых строк.
func (c *Checker) reportChanges(statuses map[string]string, errs map[string]error) {
	names := make([]string, 0, len(statuses))
	for name := range statuses {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		status := statuses[name]
		prev, seen := c.last[name]
		c.last[name] = status

		if seen && prev == status {
			continue
		}

		args := []any{"service", name, "to", status}
		if seen {
			args = append(args, "from", prev)
		}
		if err := errs[name]; err != nil {
			args = append(args, "err", err)
		}

		if status == healthpb.HealthCheckResponse_SERVING.String() {
			c.log.Info("status changed", args...)
			continue
		}
		c.log.Warn("status changed", args...)
	}
}
