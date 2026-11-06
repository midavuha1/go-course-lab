package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func mustString(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required env %s is not set", key)
	}
	return v, nil
}

func mustDuration(key string) (time.Duration, error) {
	s, err := mustString(key)
	if err != nil {
		return 0, err
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("env %s: %w", key, err)
	}

	return d, nil
}

func mustInt32(key string) (int32, error) {
	s, err := mustString(key)
	if err != nil {
		return 0, err
	}

	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("env %s: %w", key, err)
	}

	return int32(v), nil
}

func (c Config) validate() error {
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %v", c.ShutdownTimeout)
	}
	if c.DatabaseMaxConnLifetime <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONN_LIFETIME must be positive, got %v", c.DatabaseMaxConnLifetime)
	}
	if c.DatabaseConnectTimeout <= 0 {
		return fmt.Errorf("DATABASE_CONNECT_TIMEOUT must be positive, got %v", c.DatabaseConnectTimeout)
	}
	if c.DatabaseQueryTimeout <= 0 {
		return fmt.Errorf("DATABASE_QUERY_TIMEOUT must be positive, got %v", c.DatabaseQueryTimeout)
	}
	if c.DatabaseMaxConns <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be positive, got %d", c.DatabaseMaxConns)
	}
	if c.DatabaseMinConns < 0 {
		return fmt.Errorf("DATABASE_MIN_CONNS must be non-negative, got %d", c.DatabaseMinConns)
	}
	if c.DatabaseMinConns > c.DatabaseMaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS (%d) cannot be greater than DATABASE_MAX_CONNS (%d)", c.DatabaseMinConns, c.DatabaseMaxConns)
	}
	return nil
}

func Load() (Config, error) {
	var cfg Config
	var err error

	cfg.HTTPAddr, err = mustString("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = mustDuration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseURL, err = mustString("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMaxConns, err = mustInt32("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMinConns, err = mustInt32("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMaxConnLifetime, err = mustDuration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseConnectTimeout, err = mustDuration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseQueryTimeout, err = mustDuration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.LogLevel = os.Getenv("LOG_LEVEL")
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
