package config

import (
	"fmt"
	"os"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// Config — конфигурация Edge. Для Этапа 2 читаем подмножество полей
// из configs/edge.yaml простым парсером без внешних зависимостей.
// На Этапе 3+ заменим на полноценный YAML-парсер.
type Config struct {
	ClientID string

	PollInterval time.Duration

	ScenarioPath  string
	ScenarioSpeed float64
	ScenarioLoop  bool

	StoragePath string

	BackendURL     string
	SyncInterval   time.Duration
	RequestTimeout time.Duration
	BatchSize      int

	Thresholds domain.Thresholds
}

// Default возвращает конфигурацию по умолчанию (совпадает с configs/edge.yaml).
func Default() Config {
	return Config{
		ClientID:       "edge-01",
		PollInterval:   2 * time.Second,
		ScenarioPath:   "scenarios/day_cycle.csv",
		ScenarioSpeed:  60,
		ScenarioLoop:   true,
		StoragePath:    "data/edge.db",
		BackendURL:     "http://localhost:8080",
		SyncInterval:   10 * time.Second,
		RequestTimeout: 5 * time.Second,
		BatchSize:      100,
		Thresholds: domain.Thresholds{
			TempLow:           22.0,
			TempLowHysteresis: 1.0,
			TempCriticalLow:   18.0,
			TempCriticalHigh:  30.0,
			O2Low:             5.0,
			O2LowHysteresis:   0.5,
			FlowMin:           5.0,
		},
	}
}

// Load читает конфигурацию. Пока поддерживает только путь к файлу сценария
// через переменные окружения для гибкости отладки.
func Load() (Config, error) {
	cfg := Default()

	if v := os.Getenv("AQUAMIND_SCENARIO"); v != "" {
		cfg.ScenarioPath = v
	}
	if v := os.Getenv("AQUAMIND_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid AQUAMIND_POLL_INTERVAL: %w", err)
		}
		cfg.PollInterval = d
	}
	if v := os.Getenv("AQUAMIND_STORAGE"); v != "" {
		cfg.StoragePath = v
	}
	if v := os.Getenv("AQUAMIND_BACKEND_URL"); v != "" {
		cfg.BackendURL = v
	}
	if v := os.Getenv("AQUAMIND_SYNC_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid AQUAMIND_SYNC_INTERVAL: %w", err)
		}
		cfg.SyncInterval = d
	}

	if err := cfg.Thresholds.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid thresholds: %w", err)
	}
	return cfg, nil
}
