package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
}

type HTTPConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type PostgresConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Addr:         envString("HTTP_ADDR", ":8080"),
			ReadTimeout:  envDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: envDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  envDuration("HTTP_IDLE_TIMEOUT", time.Minute),
		},
		Postgres: PostgresConfig{
			DSN:             os.Getenv("DATABASE_URL"),
			MaxConns:        int32(envInt("POSTGRES_MAX_CONNS", 10)),
			MinConns:        int32(envInt("POSTGRES_MIN_CONNS", 2)),
			MaxConnLifetime: envDuration("POSTGRES_MAX_CONN_LIFETIME", time.Hour),
		},
	}

	if cfg.Postgres.DSN == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
