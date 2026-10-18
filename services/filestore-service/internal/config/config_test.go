package config

import (
	"math"
	"strings"
	"testing"
	"time"

	platformconfig "github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_SERVER_HOST       = "localhost"
	TEST_SERVER_PORT       = "50054"
	TEST_DB_HOST           = "localhost"
	TEST_DB_PORT           = 5454
	TEST_DB_USER           = "banking_filestore"
	TEST_DB_PASSWORD       = "123"
	TEST_DB_NAME           = "banking_filestore"
	TEST_DB_SSL_MODE       = "disable"
	TEST_MINIO_ENDPOINT    = "localhost:9000"
	TEST_MINIO_ACCESS_KEY  = "minioadmin"
	TEST_MINIO_SECRET_KEY  = "minioadmin"
	TEST_MINIO_BUCKET      = "kyc-filestore"
	TEST_CONNECT_TIMEOUT   = 5 * time.Second
	TEST_HEALTH_CHECK_FREQ = 15 * time.Second
	TEST_REQUEST_TIMEOUT   = 30 * time.Second
	TEST_ERR_PREFIX        = "config value is required: "
)

// validConfig - базовый корректный конфиг из configs/config.yml, каждый вызов отдаёт новое значение
func validConfig() Config {
	return Config{
		Server: platformconfig.Server{Host: TEST_SERVER_HOST, Port: TEST_SERVER_PORT},
		Postgres: platformconfig.DataBaseConfig{
			Host:     TEST_DB_HOST,
			Port:     TEST_DB_PORT,
			User:     TEST_DB_USER,
			Password: TEST_DB_PASSWORD,
			DBName:   TEST_DB_NAME,
			SSLMode:  TEST_DB_SSL_MODE,
		},
		MinIOConfig: MinIOConfig{
			Endpoint:        TEST_MINIO_ENDPOINT,
			AccessKey:       TEST_MINIO_ACCESS_KEY,
			SecretKey:       TEST_MINIO_SECRET_KEY,
			BucketName:      TEST_MINIO_BUCKET,
			ConnectTimeout:  TEST_CONNECT_TIMEOUT,
			HealthCheckFreq: TEST_HEALTH_CHECK_FREQ,
		},
		RequestTimeout: TEST_REQUEST_TIMEOUT,
	}
}

// zeroTimeouts обнуляет все таймауты конфига
func zeroTimeouts(c *Config) {
	c.RequestTimeout = 0
	c.MinIOConfig.ConnectTimeout = 0
	c.MinIOConfig.HealthCheckFreq = 0
	c.Postgres.MaxConnLifetime = 0
	c.Postgres.MaxConnIdleTime = 0
	c.Postgres.HealthCheckPeriod = 0
}

// clearDatabase обнуляет все обязательные поля группы database
func clearDatabase(c *Config) {
	c.Postgres.Host = ""
	c.Postgres.Port = 0
	c.Postgres.User = ""
	c.Postgres.DBName = ""
}

// clearMinIO обнуляет все обязательные поля группы minio_config
func clearMinIO(c *Config) {
	c.MinIOConfig.Endpoint = ""
	c.MinIOConfig.AccessKey = ""
	c.MinIOConfig.SecretKey = ""
	c.MinIOConfig.BucketName = ""
}

// keyFromErr вырезает ключ конфига из текста ошибки
func keyFromErr(t *testing.T, err error) string {
	t.Helper()
	msg := err.Error()
	require.True(t, strings.HasPrefix(msg, TEST_ERR_PREFIX), "unexpected error text: %q", msg)
	return strings.TrimPrefix(msg, TEST_ERR_PREFIX)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantKey string
	}{
		{name: "base config is valid", mutate: func(c *Config) {}},
		{name: "empty server.port", mutate: func(c *Config) { c.Server.Port = "" }, wantKey: "server.port"},
		{name: "empty database.host", mutate: func(c *Config) { c.Postgres.Host = "" }, wantKey: "database.host"},
		{name: "empty database.user", mutate: func(c *Config) { c.Postgres.User = "" }, wantKey: "database.user"},
		{name: "empty database.dbname", mutate: func(c *Config) { c.Postgres.DBName = "" }, wantKey: "database.dbname"},
		{name: "empty minio_config.endpoint", mutate: func(c *Config) { c.MinIOConfig.Endpoint = "" }, wantKey: "minio_config.endpoint"},
		{name: "empty minio_config.access_key", mutate: func(c *Config) { c.MinIOConfig.AccessKey = "" }, wantKey: "minio_config.access_key"},
		{name: "empty minio_config.secret_key", mutate: func(c *Config) { c.MinIOConfig.SecretKey = "" }, wantKey: "minio_config.secret_key"},
		{name: "empty minio_config.bucket_name", mutate: func(c *Config) { c.MinIOConfig.BucketName = "" }, wantKey: "minio_config.bucket_name"},
		{name: "database.port zero", mutate: func(c *Config) { c.Postgres.Port = 0 }, wantKey: "database.port"},
		{name: "database.port negative one", mutate: func(c *Config) { c.Postgres.Port = -1 }, wantKey: "database.port"},
		{name: "database.port min int", mutate: func(c *Config) { c.Postgres.Port = math.MinInt }, wantKey: "database.port"},
		{name: "database.port one", mutate: func(c *Config) { c.Postgres.Port = 1 }},
		{name: "database.port 65535", mutate: func(c *Config) { c.Postgres.Port = 65535 }},
		{name: "empty server.host", mutate: func(c *Config) { c.Server.Host = "" }},
		{name: "empty database.password", mutate: func(c *Config) { c.Postgres.Password = "" }},
		{name: "empty database.ssl_mode", mutate: func(c *Config) { c.Postgres.SSLMode = "" }},
		{name: "all timeouts zero", mutate: zeroTimeouts},
		{name: "zero pool and no ssl", mutate: func(c *Config) {
			c.Postgres.MaxConns = 0
			c.Postgres.MinConns = 0
			c.MinIOConfig.UseSSL = false
		}},
		{name: "server.port before database.host", mutate: func(c *Config) {
			c.Server.Port = ""
			c.Postgres.Host = ""
		}, wantKey: "server.port"},
		{name: "database.user before minio_config.endpoint", mutate: func(c *Config) {
			c.Postgres.User = ""
			c.MinIOConfig.Endpoint = ""
		}, wantKey: "database.user"},
		{name: "server.port before minio_config.bucket_name", mutate: func(c *Config) {
			c.Server.Port = ""
			c.MinIOConfig.BucketName = ""
		}, wantKey: "server.port"},
		{name: "database.port before minio_config.secret_key", mutate: func(c *Config) {
			c.Postgres.Port = 0
			c.MinIOConfig.SecretKey = ""
		}, wantKey: "database.port"},
		{name: "zero config", mutate: func(c *Config) { *c = Config{} }, wantKey: "server.port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()

			if tt.wantKey != "" {
				require.ErrorIs(t, err, domain.ErrConfigValueRequired)
				assert.EqualError(t, err, TEST_ERR_PREFIX+tt.wantKey)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_Validate_GroupOrder(t *testing.T) {
	databaseKeys := []string{"database.host", "database.port", "database.user", "database.dbname"}
	minioKeys := []string{"minio_config.endpoint", "minio_config.access_key", "minio_config.secret_key", "minio_config.bucket_name"}

	tests := []struct {
		name      string
		mutate    func(c *Config)
		groupKeys []string
	}{
		{name: "whole database group empty", mutate: clearDatabase, groupKeys: databaseKeys},
		{name: "whole minio group empty", mutate: clearMinIO, groupKeys: minioKeys},
		{name: "database group before minio group", mutate: func(c *Config) {
			clearDatabase(c)
			clearMinIO(c)
		}, groupKeys: databaseKeys},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()

			require.ErrorIs(t, err, domain.ErrConfigValueRequired)
			assert.Contains(t, tt.groupKeys, keyFromErr(t, err))
		})
	}
}

func TestConfig_Validate_OptionalFieldsInvariant(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(c *Config)
	}{
		{name: "all optional empty", mutate: func(c *Config) {
			c.Server.Host = ""
			c.Postgres.Password = ""
			c.Postgres.SSLMode = ""
			c.Postgres.MaxConns = 0
			c.Postgres.MinConns = 0
			c.MinIOConfig.UseSSL = false
			zeroTimeouts(c)
		}},
		{name: "all optional filled", mutate: func(c *Config) {
			c.MinIOConfig.UseSSL = true
			c.Postgres.MaxConns = 10
			c.Postgres.MinConns = 2
			c.Postgres.MaxConnLifetime = time.Hour
			c.Postgres.MaxConnIdleTime = 30 * time.Minute
			c.Postgres.HealthCheckPeriod = time.Minute
		}},
		{name: "mixed empty credentials", mutate: func(c *Config) {
			c.Postgres.Password = ""
			c.Server.Host = ""
			c.Postgres.SSLMode = "require"
			c.MinIOConfig.UseSSL = true
			c.RequestTimeout = 0
			c.Postgres.MaxConns = 5
		}},
		{name: "mixed empty pool", mutate: func(c *Config) {
			c.Server.Host = ""
			c.Postgres.MaxConns = 0
			c.Postgres.MinConns = 0
			c.MinIOConfig.ConnectTimeout = 0
			c.MinIOConfig.HealthCheckFreq = time.Nanosecond
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			require.NoError(t, cfg.Validate())
		})
	}
}

func TestConfig_Validate_ErrorShapeInvariant(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(c *Config)
	}{
		{name: "empty server.port", mutate: func(c *Config) { c.Server.Port = "" }},
		{name: "database.port zero", mutate: func(c *Config) { c.Postgres.Port = 0 }},
		{name: "database.port min int", mutate: func(c *Config) { c.Postgres.Port = math.MinInt }},
		{name: "zero config", mutate: func(c *Config) { *c = Config{} }},
		{name: "database and minio groups empty", mutate: func(c *Config) {
			clearDatabase(c)
			clearMinIO(c)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()

			require.ErrorIs(t, err, domain.ErrConfigValueRequired)
			assert.True(t, strings.HasPrefix(err.Error(), TEST_ERR_PREFIX), "unexpected error text: %q", err.Error())
		})
	}
}

func TestConfig_Validate_DoesNotMutate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{name: "base config", mutate: func(c *Config) {}},
		{name: "zero timeouts stay zero", mutate: zeroTimeouts},
		{name: "invalid minio endpoint", mutate: func(c *Config) { c.MinIOConfig.Endpoint = "" }, wantErr: true},
		{name: "zero config stays zero", mutate: func(c *Config) { *c = Config{} }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)
			want := cfg

			err := cfg.Validate()

			assert.Equal(t, want, cfg)
			if tt.wantErr {
				require.ErrorIs(t, err, domain.ErrConfigValueRequired)
				return
			}
			require.NoError(t, err)
		})
	}
}
