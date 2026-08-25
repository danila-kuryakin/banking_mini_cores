package config

import (
	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

// Config is the auth-service configuration beyond the common base.
type Config struct {
	ServerAddr string
	//PostgresDSN string
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	//dsn, err := config.MustString("POSTGRES_DSN")
	//if err != nil {
	//	return Config{}, err
	//}

	return Config{
		ServerAddr: config.String("SERVER_ADDR", ":50052"),
		//PostgresDSN: dsn,
	}, nil
}
