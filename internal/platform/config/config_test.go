package config

import (
	"testing"
	"time"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "secret")
	if _, err := Load(); err == nil {
		t.Fatal("expected DATABASE_URL error")
	}
}

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://lending:lending@localhost:5432/lending?sslmode=disable")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("JWT_TTL", "2h")
	t.Setenv("WORKER_POLL_INTERVAL", "5s")
	t.Setenv("WORKER_BATCH_SIZE", "10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTP_ADDR = %s", cfg.HTTPAddr)
	}
	if cfg.JWTTTL != 2*time.Hour {
		t.Fatalf("JWT_TTL = %s", cfg.JWTTTL)
	}
	if cfg.WorkerPollInterval != 5*time.Second {
		t.Fatalf("poll interval = %s", cfg.WorkerPollInterval)
	}
	if cfg.WorkerBatchSize != 10 {
		t.Fatalf("batch size = %d", cfg.WorkerBatchSize)
	}
}

func TestRequireJWTSecret(t *testing.T) {
	cfg := Config{}
	if err := cfg.RequireJWTSecret(); err == nil {
		t.Fatal("expected JWT secret error")
	}
}
