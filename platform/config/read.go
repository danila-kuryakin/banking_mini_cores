package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Validator — необязательный интерфейс. Если *T его реализует,
// Read вызовет Validate после разбора.
type Validator interface {
	Validate() error
}

type options struct {
	name      string // имя файла без расширения
	envPrefix string
	envFile   string
	optional  bool // не падать, если config-файла нет
}

type Option func(*options)

func WithName(n string) Option      { return func(o *options) { o.name = n } }
func WithEnvPrefix(p string) Option { return func(o *options) { o.envPrefix = p } }
func WithEnvFile(f string) Option   { return func(o *options) { o.envFile = f } }
func Optional() Option              { return func(o *options) { o.optional = true } }

// Read читает <path>/config.yml, накладывает поверх .env и окружение
// процесса и разбирает результат в T.
func Read[T any](path string, opts ...Option) (*T, error) {
	o := options{name: "config", envFile: ".env"}
	for _, fn := range opts {
		fn(&o)
	}

	// 1. .env -> окружение процесса. Уже установленные переменные
	// не перезатираются: реальный ENV приоритетнее файла.
	if err := godotenv.Load(o.envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("чтение %s: %w", o.envFile, err)
	}

	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName(o.name)
	v.SetConfigType("yaml")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if o.envPrefix != "" {
		v.SetEnvPrefix(o.envPrefix)
	}
	v.AutomaticEnv()

	// 2. Привязываем ENV к каждому ключу структуры T.
	// Делаем это до ReadInConfig, чтобы ключи работали
	// независимо от наличия файла.
	var cfg T
	t := reflect.TypeOf(cfg)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config: T должен быть структурой, получено %s", t.Kind())
	}
	for _, key := range structKeys(t, "") {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf("bind env %q: %w", key, err)
		}
	}

	// 3. config.yml — база значений.
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) || !o.optional {
			return nil, fmt.Errorf("чтение конфига: %w", err)
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("разбор конфига: %w", err)
	}

	if val, ok := any(&cfg).(Validator); ok {
		if err := val.Validate(); err != nil {
			return nil, fmt.Errorf("валидация конфига: %w", err)
		}
	}
	return &cfg, nil
}

var timeType = reflect.TypeOf(time.Time{})

// structKeys собирает пути вида "database.host" из mapstructure-тегов.
func structKeys(t reflect.Type, prefix string) []string {
	var keys []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		tag := f.Tag.Get("mapstructure")
		name, rest, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToLower(f.Name)
		}

		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		// встроенная структура со ,squash живёт на уровне родителя
		if strings.Contains(rest, "squash") && ft.Kind() == reflect.Struct {
			keys = append(keys, structKeys(ft, prefix)...)
			continue
		}

		key := name
		if prefix != "" {
			key = prefix + "." + name
		}

		if ft.Kind() == reflect.Struct && ft != timeType {
			keys = append(keys, structKeys(ft, key)...)
			continue
		}
		keys = append(keys, key)
	}
	return keys
}
