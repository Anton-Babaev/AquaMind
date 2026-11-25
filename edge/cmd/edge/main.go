package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Anton-Babaev/AquaMind/edge/adapter/backendclient"
	"github.com/Anton-Babaev/AquaMind/edge/adapter/hardware/console"
	"github.com/Anton-Babaev/AquaMind/edge/adapter/hardware/gpio"
	"github.com/Anton-Babaev/AquaMind/edge/adapter/hardware/simulation"
	"github.com/Anton-Babaev/AquaMind/edge/adapter/storage/sqlite"
	"github.com/Anton-Babaev/AquaMind/edge/config"
	edgehttp "github.com/Anton-Babaev/AquaMind/edge/delivery/http"
	"github.com/Anton-Babaev/AquaMind/edge/delivery/poller"
	"github.com/Anton-Babaev/AquaMind/edge/delivery/retention"
	"github.com/Anton-Babaev/AquaMind/edge/delivery/syncworker"
	"github.com/Anton-Babaev/AquaMind/edge/delivery/watchdog"
	"github.com/Anton-Babaev/AquaMind/edge/usecase/alert"
	"github.com/Anton-Babaev/AquaMind/edge/usecase/control"
	"github.com/Anton-Babaev/AquaMind/edge/usecase/safety"
	"github.com/Anton-Babaev/AquaMind/edge/usecase/settings"
	"github.com/Anton-Babaev/AquaMind/edge/usecase/uplink"
)

// actuators — общий тип для адаптеров выходов (console и gpio).
// Оба реализуют control.Actuators; gpio дополнительно умеет Close().
type actuators interface {
	control.Actuators
}

// outboxAdapter адаптирует sqlite.Repository к порту uplink.Outbox:
// преобразует []sqlite.OutboxEntry в []uplink.OutboxEntry.
type outboxAdapter struct {
	store *sqlite.Repository
}

func (a outboxAdapter) PendingOutbox(ctx context.Context, limit int) ([]uplink.OutboxEntry, error) {
	entries, err := a.store.PendingOutbox(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]uplink.OutboxEntry, len(entries))
	for i, e := range entries {
		out[i] = uplink.OutboxEntry{ID: e.ID, Payload: e.Payload}
	}
	return out, nil
}

func (a outboxAdapter) MarkSent(ctx context.Context, ids []int64) error {
	return a.store.MarkSent(ctx, ids)
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

	// UseCase: управление (владелец выходов).
	ctrl := control.New(sensors, act, store, tracker)

	// UseCase: пороги (валидация + сохранение).
	settingsUC := settings.New(store)

	// UseCase: синхронизация с Backend.
	backend := backendclient.New(cfg.BackendURL, cfg.RequestTimeout)
	uplinkUC := uplink.New(outboxAdapter{store: store}, backend, cfg.ClientID, cfg.BatchSize)

	// UseCase: сторожевой таймер (Safe Mode по устареванию данных).
	safetyUC := safety.New(ctrl, ctrl, cfg.Safety.StaleAfter)

	// Delivery: поллер, синхронизация, watchdog, retention.
	p := poller.New(ctrl, cfg.PollInterval)
	sync := syncworker.New(uplinkUC, cfg.SyncInterval)
	wd := watchdog.New(safetyUC, cfg.Safety.Interval)
	// Очистку устаревших данных запускаем раз в час (или реже, если retention мал).
	retentionInterval := time.Hour
	if cfg.Storage.Retention > 0 && cfg.Storage.Retention < retentionInterval {
		retentionInterval = cfg.Storage.Retention
	}
	ret := retention.New(store, cfg.Storage.Retention, retentionInterval)

	// Delivery: локальный веб-интерфейс (дашборд + пороги).
	web := edgehttp.NewServer(cfg.ControlPlane.Listen, ctrl, settingsUC, store, cfg.ControlPlane.AllowedCIDRs)

	// --- Запуск ---

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("edge started: client_id=%s hardware=%s scenario=%s interval=%s backend=%s",
		cfg.ClientID, cfg.Hardware, cfg.ScenarioPath, cfg.PollInterval, cfg.BackendURL)

	// SyncWorker в отдельной горутине.
	go sync.Run(ctx)

	// Watchdog (Safe Mode) в отдельной горутине.
	go wd.Run(ctx)

	// Очистка устаревших данных в отдельной горутине.
	go ret.Run(ctx)

	// Веб-интерфейс в отдельной горутине.
	go func() {
		if err := web.Run(ctx); err != nil {
			log.Printf("http server: %v", err)
		}
	}()

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
