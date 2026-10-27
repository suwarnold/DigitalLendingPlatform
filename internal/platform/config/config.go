package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	JWTTTL             time.Duration
	HTTPAddr           string
	LogLevel           string
	WorkerPollInterval time.Duration
	WorkerBatchSize    int
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTTTL:             24 * time.Hour,
		HTTPAddr:           ":8080",
		LogLevel:           "info",
		WorkerPollInterval: 2 * time.Second,
		WorkerBatchSize:    50,
	}

	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := os.Getenv("JWT_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("JWT_TTL: %w", err)
		}
		cfg.JWTTTL = d
	}
	if v := os.Getenv("WORKER_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("WORKER_POLL_INTERVAL: %w", err)
		}
		cfg.WorkerPollInterval = d
	}
	if v := os.Getenv("WORKER_BATCH_SIZE"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("WORKER_BATCH_SIZE must be a positive integer")
		}
		cfg.WorkerBatchSize = n
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func (c Config) RequireJWTSecret() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}
