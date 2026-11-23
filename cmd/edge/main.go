package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/backendclient"
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/hardware/console"
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/hardware/simulation"
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/storage/sqlite"
	"github.com/Anton-Babaev/AquaMind/internal/edge/config"
	"github.com/Anton-Babaev/AquaMind/internal/edge/delivery/poller"
	"github.com/Anton-Babaev/AquaMind/internal/edge/delivery/syncworker"
	"github.com/Anton-Babaev/AquaMind/internal/edge/usecase/alert"
	"github.com/Anton-Babaev/AquaMind/internal/edge/usecase/control"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// --- Сборка зависимостей (DI) ---

	// HAL: источник показаний (CSV-симуляция) и выходы (console-заглушка).
	sensors, err := simulation.NewProvider(cfg.ScenarioPath, cfg.ScenarioSpeed, cfg.ScenarioLoop)
	if err != nil {
		log.Fatalf("simulation provider: %v", err)
	}
	actuators := console.New()

	// Хранилище: SQLite (WAL).
	if err := os.MkdirAll(filepath.Dir(cfg.StoragePath), 0o755); err != nil {
		log.Fatalf("create storage dir: %v", err)
	}
	store, err := sqlite.Open(cfg.StoragePath)
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	// Инициализируем пороги в БД, если их там ещё нет.
	if err := store.SaveThresholds(context.Background(), cfg.Thresholds); err != nil {
		log.Fatalf("init thresholds: %v", err)
	}

	// Трекер алертов.
	tracker := alert.NewTracker()

	// UseCase.
	ctrl := control.New(sensors, actuators, store, tracker)

	// Delivery: поллер и синхронизация.
	p := poller.New(ctrl, cfg.PollInterval)
	backend := backendclient.New(cfg.BackendURL, cfg.RequestTimeout)
	sync := syncworker.New(store, backend, cfg.ClientID, cfg.SyncInterval, cfg.BatchSize)

	// --- Запуск ---

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("edge started: client_id=%s scenario=%s interval=%s backend=%s",
		cfg.ClientID, cfg.ScenarioPath, cfg.PollInterval, cfg.BackendURL)

	// SyncWorker в отдельной горутине.
	go sync.Run(ctx)

	// Graceful shutdown: по сигналу отменяем контекст, поллер останавливается.
	p.Run(ctx)

	log.Printf("edge stopped")
}
