package config

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Auth       AuthConfig
	Metrics    MetricsConfig
	ClickHouse ClickHouseConfig
}

type ServerConfig struct {
	Port int    `env:"BACKEND_PORT, default=8080"`
	Host string `env:"BACKEND_HOST, default=0.0.0.0"`
}

type DatabaseConfig struct {
	Host     string `env:"POSTGRES_HOST, default=localhost"`
	Port     int    `env:"POSTGRES_PORT, default=5432"`
	User     string `env:"POSTGRES_USER, default=containerscope"`
	Password string `env:"POSTGRES_PASSWORD, default=containerscope"`
	Name     string `env:"POSTGRES_DB, default=containerscope"`
	SSLMode  string `env:"POSTGRES_SSLMODE, default=disable"`
}

type AuthConfig struct {
	JWTSecret  string        `env:"JWT_SECRET, default=change-me-in-production"`
	AccessTTL  time.Duration `env:"JWT_ACCESS_TTL, default=15m"`
	RefreshTTL time.Duration `env:"JWT_REFRESH_TTL, default=168h"`
}

type MetricsConfig struct {
	VictoriaMetricsURL string `env:"VICTORIAMETRICS_URL, default=http://localhost:8428"`
}

type ClickHouseConfig struct {
	Host     string `env:"CLICKHOUSE_HOST, default=localhost"`
	Port     int    `env:"CLICKHOUSE_PORT, default=9000"`
	Database string `env:"CLICKHOUSE_DB, default=containerscope"`
	User     string `env:"CLICKHOUSE_USER, default=containerscope"`
	Password string `env:"CLICKHOUSE_PASSWORD, default=containerscope"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return nil
}
