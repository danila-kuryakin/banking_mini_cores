package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Base is the configuration every service has, whatever it does.
type Base struct {
	// ServiceName имя сервера
	ServiceName string

	// Env dev, staging или prod. Для выдачи логов gRGC сервера
	Env string

	// LogLevel debug, info, warn или error.
	LogLevel string

	// GRPCAddr адрес gRPC сервера.
	GRPCAddr string

	// ShutdownTimeout время соединения.
	ShutdownTimeout time.Duration
}

// IsDev сообщает, работает ли служба в режиме dev.
func (b Base) IsDev() bool { return b.Env == "dev" }

func LoadBase(serviceName string, defaultGRPCPort int) (Base, error) {

	shutdown, err := Duration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Base{}, err
	}

	return Base{
		ServiceName:     String("SERVICE_NAME", serviceName),
		Env:             String("ENV", "dev"),
		LogLevel:        String("LOG_LEVEL", "info"),
		GRPCAddr:        String("GRPC_ADDR", fmt.Sprintf(":%d", defaultGRPCPort)),
		ShutdownTimeout: shutdown,
	}, nil
}

// String возвращает значение ключа или def, если ключ не задан или пуст.
func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}

	return def
}

// MustString возвращает значение ключа или выдает ошибку, если ключ не задан.
// Используйте этот метод для всего, для чего сервис не может подобрать
// безопасное значение по умолчанию, например для DSN базы данных или
// ключа подписи.
func MustString(key string) (string, error) {
	v := String(key, "")
	if v == "" {
		return "", fmt.Errorf("config: required environment variable %s is not set", key)
	}

	return v, nil
}

// Int возвращает значение ключа, преобразованное в целое число, или def,
// если значение не задано.
func Int(key string, def int) (int, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer, got %q: %w", key, raw, err)
	}

	return v, nil
}

// Float возвращает значение ключа, преобразованное в число с плавающей
// запятой, или def, если значение не задано.
func Float(key string, def float64) (float64, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a number, got %q: %w", key, raw, err)
	}

	return v, nil
}

// Bool возвращает значение ключа, проанализированное как логическое,
// или def, если значение не задано.
func Bool(key string, def bool) (bool, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("config: %s must be a boolean, got %q: %w", key, raw, err)
	}

	return v, nil
}

// Duration возвращает значение ключа, преобразованное в
// продолжительность в формате Go, например «15s», или def,
// если значение не задано.
func Duration(key string, def time.Duration) (time.Duration, error) {
	raw := String(key, "")
	if raw == "" {
		return def, nil
	}

	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a duration, got %q: %w", key, raw, err)
	}

	return v, nil
}

// StringSlice возвращает значение ключа, разделенного
// запятыми, или значение по умолчанию, если ключ не задан.
func StringSlice(key string, def []string) []string {
	raw := String(key, "")
	if raw == "" {
		return def
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return def
	}

	return out
}
