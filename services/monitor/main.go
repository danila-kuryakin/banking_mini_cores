// Команда monitor периодически опрашивает health-проверки всех сервисов
// (grpc.health.v1.Health) и печатает их состояние в консоль.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/monitor/internal/checker"
)

const defaultTargets = "" +
	"auth=localhost:50051, " +
	"customer=localhost:50052, " +
	"kyc=localhost:50053, " +
	"document=localhost:50054, " +
	"account=localhost:50055, " +
	"ledger=localhost:50056, " +
	"antifraud=localhost:50057, " +
	"notification=localhost:50058"

func main() {
	targets := flag.String("targets", envOr("MONITOR_TARGETS", defaultTargets), "список сервисов вида имя=адрес через запятую")
	interval := flag.Duration("interval", envDurationOr("MONITOR_INTERVAL", 5*time.Second), "период опроса")
	timeout := flag.Duration("timeout", envDurationOr("MONITOR_TIMEOUT", 2*time.Second), "таймаут одного круга опроса")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(*targets, *interval, *timeout, log); err != nil {
		log.Error("monitor failed", "err", err)
		os.Exit(1)
	}
}

func run(targets string, interval, timeout time.Duration, log *slog.Logger) error {
	parsed, err := parseTargets(targets)
	if err != nil {
		return err
	}

	c, err := checker.New(log, parsed)
	if err != nil {
		return err
	}
	defer c.Close()

	names := make([]string, 0, len(parsed))
	for _, t := range parsed {
		names = append(names, t.Name+"="+t.Addr)
	}
	log.Info("monitor started", "interval", interval, "timeout", timeout, "targets", strings.Join(names, " "))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// Первый круг делаем сразу, не дожидаясь тика.
		checkOnce(c, timeout)

		select {
		case <-ticker.C:
		case <-stop:
			log.Info("shutting down")
			return nil
		}
	}
}

func checkOnce(c *checker.Checker, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c.CheckAll(ctx)
}

// parseTargets разбирает строку.
func parseTargets(raw string) ([]checker.Target, error) {
	var targets []checker.Target

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name, addr, ok := strings.Cut(part, "=")
		if !ok || name == "" || addr == "" {
			return nil, fmt.Errorf("цель %q: ожидался формат имя=адрес", part)
		}

		targets = append(targets, checker.Target{Name: name, Addr: addr})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("список целей пуст")
	}

	return targets, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}

	return d
}
