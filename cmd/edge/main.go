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
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/hardware/gpio"
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/hardware/simulation"
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/storage/sqlite"
	"github.com/Anton-Babaev/AquaMind/internal/edge/config"
	"github.com/Anton-Babaev/AquaMind/internal/edge/delivery/poller"
	"github.com/Anton-Babaev/AquaMind/internal/edge/delivery/syncworker"
	"github.com/Anton-Babaev/AquaMind/internal/edge/usecase/alert"
	"github.com/Anton-Babaev/AquaMind/internal/edge/usecase/control"
)

// actuators — общий тип для адаптеров выходов (console и gpio).
// Оба реализуют control.Actuators; gpio дополнительно умеет Close().
type actuators interface {
	control.Actuators
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Загружаем конфигурацию: флаг --config <путь> (из systemd-юнита),
	// иначе — дефолты + переменные окружения AQUAMIND_*.
	cfg, err := config.LoadFromArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// --- Сборка зависимостей (DI) ---

	// HAL: источник показаний (CSV-симуляция).
	sensors, err := simulation.NewProvider(cfg.ScenarioPath, cfg.ScenarioSpeed, cfg.ScenarioLoop)
	if err != nil {
		log.Fatalf("simulation provider: %v", err)
	}

	// HAL: выходы — выбор адаптера по конфигу (hardware: console|gpio).
	act, closeActuators, err := buildActuators(cfg)
	if err != nil {
		log.Fatalf("actuators: %v", err)
	}
	defer closeActuators()

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
	ctrl := control.New(sensors, act, store, tracker)

	// Delivery: поллер и синхронизация.
	p := poller.New(ctrl, cfg.PollInterval)
	backend := backendclient.New(cfg.BackendURL, cfg.RequestTimeout)
	sync := syncworker.New(store, backend, cfg.ClientID, cfg.SyncInterval, cfg.BatchSize)

	// --- Запуск ---

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("edge started: client_id=%s hardware=%s scenario=%s interval=%s backend=%s",
		cfg.ClientID, cfg.Hardware, cfg.ScenarioPath, cfg.PollInterval, cfg.BackendURL)

	// SyncWorker в отдельной горутине.
	go sync.Run(ctx)

	// Graceful shutdown: по сигналу отменяем контекст, поллер останавливается.
	p.Run(ctx)

	log.Printf("edge stopped")
}

// buildActuators создаёт адаптер выходов по конфигу и возвращает функцию
// освобождения ресурсов (для gpio — гасит светодиоды и освобождает GPIO).
func buildActuators(cfg config.Config) (actuators, func(), error) {
	switch cfg.Hardware {
	case config.HardwareGPIO:
		leds, err := gpio.New(gpio.Pins{
			Heating: cfg.GPIOPins.Heating,
			Alarm:   cfg.GPIOPins.Alarm,
			Normal:  cfg.GPIOPins.Normal,
		})
		if err != nil {
			return nil, nil, err
		}
		log.Printf("hardware: gpio (BCM heating=%d alarm=%d normal=%d)",
			cfg.GPIOPins.Heating, cfg.GPIOPins.Alarm, cfg.GPIOPins.Normal)
		return leds, func() {
			if err := leds.Close(); err != nil {
				log.Printf("gpio close: %v", err)
			}
		}, nil

	case config.HardwareConsole:
		fallthrough
	default:
		log.Printf("hardware: console")
		return console.New(), func() {}, nil
	}
}
