package config

import "net"

// Base базовая настройка сервера
type Server struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

func (s Server) GetAddr() string {
	return net.JoinHostPort(s.Host, s.Port)
}
