package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault_Valid(t *testing.T) {
	cfg := Default()
	if err := cfg.validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("AQUAMIND_BACKEND_LISTEN", ":9999")
	t.Setenv("AQUAMIND_BACKEND_POSTGRES_DSN", "postgres://x")
	t.Setenv("AQUAMIND_BACKEND_TG_TOKEN", "tok")
	t.Setenv("AQUAMIND_BACKEND_TG_CHAT_ID", "42")
	t.Setenv("AQUAMIND_BACKEND_NOTIFY_COOLDOWN", "1m")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Listen != ":9999" {
		t.Fatalf("want :9999, got %s", cfg.Listen)
	}
	if cfg.Postgres.DSN != "postgres://x" {
		t.Fatalf("want dsn override, got %s", cfg.Postgres.DSN)
	}
	if !cfg.Telegram.Enabled {
		t.Fatal("token env should enable telegram")
	}
	if cfg.Notify.Cooldown != time.Minute {
		t.Fatalf("want 1m cooldown, got %v", cfg.Notify.Cooldown)
	}
}

func TestLoad_YAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backend.yaml")
	yaml := `
listen: ":7777"
postgres:
  dsn: "postgres://file"
  max_conns: 5
telegram:
  enabled: false
notify:
  cooldown: "2m"
  notify_cleared: false
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Listen != ":7777" {
		t.Fatalf("want :7777, got %s", cfg.Listen)
	}
	if cfg.Postgres.DSN != "postgres://file" {
		t.Fatalf("want file dsn, got %s", cfg.Postgres.DSN)
	}
	if cfg.Postgres.MaxConns != 5 {
		t.Fatalf("want max_conns 5, got %d", cfg.Postgres.MaxConns)
	}
	if cfg.Notify.Cooldown != 2*time.Minute {
		t.Fatalf("want 2m, got %v", cfg.Notify.Cooldown)
	}
	if cfg.Notify.NotifyCleared {
		t.Fatal("want notify_cleared=false")
	}
}

func TestValidate_TelegramEnabledWithoutToken(t *testing.T) {
	cfg := Default()
	cfg.Telegram.Enabled = true
	cfg.Telegram.Token = ""
	if err := cfg.validate(); err == nil {
		t.Fatal("want error: telegram enabled without token")
	}
}
