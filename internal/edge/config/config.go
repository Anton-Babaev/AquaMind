// Package config отвечает за загрузку конфигурации Edge-контроллера.
//
// Источники настроек (в порядке возрастания приоритета):
//  1. Значения по умолчанию (функция Default) — всегда есть, даже без файла.
//  2. YAML-файл (configs/edge.yaml) — накладывается поверх дефолтов.
//  3. Переменные окружения AQUAMIND_* — переопределяют всё (удобно для отладки
//     и для systemd, где настройки можно задать через Environment=).
//
// Такой порядок позволяет: держать «нормальные» настройки в файле, а точечно
// менять их при запуске, не редактируя файл.
package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
	"gopkg.in/yaml.v3"
)

// HardwareKind — тип адаптера выходов (светодиодов).
type HardwareKind string

const (
	// HardwareConsole — «лампочки» в терминале (разработка на ноутбуке).
	HardwareConsole HardwareKind = "console"
	// HardwareGPIO — реальные светодиоды на Raspberry Pi через go-rpio.
	HardwareGPIO HardwareKind = "gpio"
)

// Config — итоговая (уже собранная) конфигурация Edge, с которой работает
// остальной код. Здесь все значения приведены к «удобным» типам Go
// (time.Duration, float64 и т.д.), а не к строкам из YAML.
type Config struct {
	ClientID string // идентификатор устройства (уходит на Backend)

	PollInterval time.Duration // как часто опрашивать датчики

	ScenarioPath  string  // путь к CSV-сценарию (имитация датчиков)
	ScenarioSpeed float64 // ускорение времени сценария (1 сек реального = N сек сценария)
	ScenarioLoop  bool    // зацикливать ли сценарий

	StoragePath string // путь к файлу SQLite

	BackendURL     string        // адрес Backend (куда досылать данные)
	SyncInterval   time.Duration // как часто отправлять накопленные данные
	RequestTimeout time.Duration // таймаут одного HTTP-запроса к Backend
	BatchSize      int           // максимум записей в одном пакете синхронизации

	// Hardware — какой адаптер выходов использовать: console|gpio.
	Hardware HardwareKind
	// GPIOPins — раскладка BCM-пинов (используется только при Hardware=gpio).
	GPIOPins GPIOPins

	// Thresholds — пороги и гистерезис правил управления.
	Thresholds domain.Thresholds

	// --- Поля ниже пока читаются из файла, но ещё не используются в логике. ---
	// Оставлены, чтобы YAML-парсер был полным и файл не «терял» данные.
	// Подключаются на будущих этапах (watchdog, retention, HTTP control plane).

	// Safety — настройки сторожевого таймера (Safe Mode по устареванию данных).
	Safety SafetyConfig
	// Storage — дополнительные параметры SQLite (busy_timeout, retention).
	Storage StorageConfig
	// ControlPlane — локальный HTTP-сервер управления (пороги, настройки).
	ControlPlane ControlPlaneConfig
}

// GPIOPins — номера BCM-пинов светодиодов (см. docs/PROGRESS.md).
type GPIOPins struct {
	Heating int // 🟡 подогрев, GPIO17 / пин 11
	Alarm   int // 🔴 тревога,  GPIO27 / пин 13
	Normal  int // 🟢 норма,    GPIO22 / пин 15
}

// SafetyConfig — параметры SafetyWatchdog (пока не подключён к логике).
type SafetyConfig struct {
	Interval   time.Duration // как часто проверять «свежесть» данных
	StaleAfter time.Duration // данные старше этого — переход в Safe Mode
}

// StorageConfig — дополнительные параметры хранилища SQLite.
type StorageConfig struct {
	BusyTimeout time.Duration // сколько ждать освобождения БД при блокировке
	Retention   time.Duration // сколько хранить уже отправленные записи
}

// ControlPlaneConfig — настройки локального HTTP-сервера управления.
type ControlPlaneConfig struct {
	Listen       string   // адрес прослушивания, например ":8081"
	AllowedCIDRs []string // подсети, которым разрешён доступ (только LAN)
}

// Default возвращает конфигурацию по умолчанию.
// Значения совпадают с configs/edge.yaml — так программа работает даже без файла.
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
		Hardware:       HardwareConsole,
		GPIOPins: GPIOPins{
			Heating: 17,
			Alarm:   27,
			Normal:  22,
		},
		Thresholds: domain.Thresholds{
			TempLow:           22.0,
			TempLowHysteresis: 1.0,
			TempCriticalLow:   18.0,
			TempCriticalHigh:  30.0,
			O2Low:             5.0,
			O2LowHysteresis:   0.5,
			FlowMin:           5.0,
		},
		Safety: SafetyConfig{
			Interval:   1 * time.Second,
			StaleAfter: 10 * time.Second,
		},
		Storage: StorageConfig{
			BusyTimeout: 5 * time.Second,
			Retention:   24 * time.Hour,
		},
		ControlPlane: ControlPlaneConfig{
			Listen: ":8081",
			AllowedCIDRs: []string{
				"127.0.0.1/32",
				"192.168.0.0/16",
				"10.0.0.0/8",
			},
		},
	}
}

// Load собирает конфигурацию из трёх источников (см. комментарий к пакету):
// дефолты → YAML-файл (если path != "") → переменные окружения AQUAMIND_*.
//
// Если path пустой, файл не читается (режим «только дефолты + env»,
// удобен для быстрого запуска на ноутбуке).
func Load(path string) (Config, error) {
	cfg := Default()

	// 1. Накладываем значения из YAML-файла (если он задан).
	if path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return cfg, err
		}
	}

	// 2. Переопределяем переменными окружения (высший приоритет).
	if err := applyEnv(&cfg); err != nil {
		return cfg, err
	}

	// 3. Проверяем итоговую конфигурацию на корректность.
	if err := cfg.validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadFromArgs разбирает аргументы командной строки и загружает конфигурацию.
// Поддерживается флаг --config <путь к YAML>. Если флаг не задан — работает
// как Load("") (дефолты + env).
//
// Используется в cmd/edge/main.go, чтобы systemd-юнит мог передать путь:
//
//	ExecStart=/opt/aquamind/edge-pi --config /opt/aquamind/configs/edge.yaml
func LoadFromArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("edge", flag.ContinueOnError)
	configPath := fs.String("config", "", "путь к YAML-файлу конфигурации (например, configs/edge.yaml)")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	return Load(*configPath)
}

// loadFile читает YAML-файл по пути path и накладывает его значения на cfg.
// Поля, отсутствующие в файле, остаются такими, какими были в cfg (дефолты).
func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}

	// Разбираем YAML в «сырую» структуру (см. fileConfig ниже).
	var raw fileConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse yaml %q: %w", path, err)
	}

	// Переносим заполненные поля из «сырой» структуры в итоговую Config.
	raw.applyTo(cfg)
	return nil
}

// applyEnv переопределяет поля конфигурации значениями из переменных
// окружения AQUAMIND_*. Пустые переменные игнорируются.
func applyEnv(cfg *Config) error {
	if v := os.Getenv("AQUAMIND_SCENARIO"); v != "" {
		cfg.ScenarioPath = v
	}
	if v := os.Getenv("AQUAMIND_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid AQUAMIND_POLL_INTERVAL: %w", err)
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
			return fmt.Errorf("invalid AQUAMIND_SYNC_INTERVAL: %w", err)
		}
		cfg.SyncInterval = d
	}
	if v := os.Getenv("AQUAMIND_HARDWARE"); v != "" {
		cfg.Hardware = HardwareKind(v)
	}
	return nil
}

// validate проверяет корректность итоговой конфигурации.
func (c Config) validate() error {
	switch c.Hardware {
	case HardwareConsole, HardwareGPIO:
	default:
		return fmt.Errorf("invalid hardware %q: want %q or %q", c.Hardware, HardwareConsole, HardwareGPIO)
	}

	if err := c.Thresholds.Validate(); err != nil {
		return fmt.Errorf("invalid thresholds: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// «Сырые» структуры для разбора YAML.
//
// Они повторяют структуру configs/edge.yaml и нужны, потому что:
//   - в YAML длительности записаны строками ("10s"), а не числом наносекунд;
//   - вложенность файла (backend:, polling: и т.д.) не совпадает с плоской Config.
//
// Поля — указатели (*int, *float64, *duration): nil означает «в файле не задано»,
// и тогда мы НЕ трогаем соответствующее поле в Config (остаётся дефолт).
// ---------------------------------------------------------------------------

// fileConfig — корневая структура YAML-файла.
type fileConfig struct {
	ClientID     string            `yaml:"client_id"`
	Hardware     string            `yaml:"hardware"`
	GPIO         *fileGPIO         `yaml:"gpio"`
	Backend      *fileBackend      `yaml:"backend"`
	Polling      *filePolling      `yaml:"polling"`
	Safety       *fileSafety       `yaml:"safety"`
	Scenario     *fileScenario     `yaml:"scenario"`
	Storage      *fileStorage      `yaml:"storage"`
	ControlPlane *fileControlPlane `yaml:"control_plane"`
	Thresholds   *fileThresholds   `yaml:"thresholds"`
}

// fileGPIO — секция gpio: (раскладка пинов).
type fileGPIO struct {
	Heating *int `yaml:"heating"`
	Alarm   *int `yaml:"alarm"`
	Normal  *int `yaml:"normal"`
}

// fileBackend — секция backend: (адрес и параметры синхронизации).
type fileBackend struct {
	URL            *string   `yaml:"url"`
	SyncInterval   *duration `yaml:"sync_interval"`
	RequestTimeout *duration `yaml:"request_timeout"`
	BatchSize      *int      `yaml:"batch_size"`
}

// filePolling — секция polling: (частота опроса датчиков).
type filePolling struct {
	Interval *duration `yaml:"interval"`
}

// fileSafety — секция safety: (параметры watchdog).
type fileSafety struct {
	Interval   *duration `yaml:"interval"`
	StaleAfter *duration `yaml:"stale_after"`
}

// fileScenario — секция scenario: (имитация датчиков из CSV).
type fileScenario struct {
	Path  *string  `yaml:"path"`
	Speed *float64 `yaml:"speed"`
	Loop  *bool    `yaml:"loop"`
}

// fileStorage — секция storage: (параметры SQLite).
type fileStorage struct {
	Path        *string   `yaml:"path"`
	BusyTimeout *duration `yaml:"busy_timeout"`
	Retention   *duration `yaml:"retention"`
}

// fileControlPlane — секция control_plane: (локальный HTTP-сервер).
type fileControlPlane struct {
	Listen       *string  `yaml:"listen"`
	AllowedCIDRs []string `yaml:"allowed_cidrs"`
}

// fileThresholds — секция thresholds: (пороги правил).
type fileThresholds struct {
	TempLow           *float64 `yaml:"temp_low"`
	TempLowHysteresis *float64 `yaml:"temp_low_hysteresis"`
	TempCriticalLow   *float64 `yaml:"temp_critical_low"`
	TempCriticalHigh  *float64 `yaml:"temp_critical_high"`
	O2Low             *float64 `yaml:"o2_low"`
	O2LowHysteresis   *float64 `yaml:"o2_low_hysteresis"`
	FlowMin           *float64 `yaml:"flow_min"`
}

// duration — обёртка над time.Duration для разбора строк вида "10s", "500ms".
// yaml.v3 не умеет парсить time.Duration из строки «из коробки», поэтому
// реализуем интерфейс yaml.Unmarshaler.
type duration time.Duration

// UnmarshalYAML вызывается yaml.v3 при разборе значения в поле типа duration.
func (d *duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("duration must be a string like \"10s\": %w", err)
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = duration(parsed)
	return nil
}

// applyTo переносит заданные (не nil) поля из «сырой» структуры в итоговую Config.
// Незаданные поля остаются нетронутыми (сохраняются дефолты).
func (f fileConfig) applyTo(cfg *Config) {
	if f.ClientID != "" {
		cfg.ClientID = f.ClientID
	}
	if f.Hardware != "" {
		cfg.Hardware = HardwareKind(f.Hardware)
	}

	if f.GPIO != nil {
		if f.GPIO.Heating != nil {
			cfg.GPIOPins.Heating = *f.GPIO.Heating
		}
		if f.GPIO.Alarm != nil {
			cfg.GPIOPins.Alarm = *f.GPIO.Alarm
		}
		if f.GPIO.Normal != nil {
			cfg.GPIOPins.Normal = *f.GPIO.Normal
		}
	}

	if f.Backend != nil {
		if f.Backend.URL != nil {
			cfg.BackendURL = *f.Backend.URL
		}
		if f.Backend.SyncInterval != nil {
			cfg.SyncInterval = time.Duration(*f.Backend.SyncInterval)
		}
		if f.Backend.RequestTimeout != nil {
			cfg.RequestTimeout = time.Duration(*f.Backend.RequestTimeout)
		}
		if f.Backend.BatchSize != nil {
			cfg.BatchSize = *f.Backend.BatchSize
		}
	}

	if f.Polling != nil && f.Polling.Interval != nil {
		cfg.PollInterval = time.Duration(*f.Polling.Interval)
	}

	if f.Safety != nil {
		if f.Safety.Interval != nil {
			cfg.Safety.Interval = time.Duration(*f.Safety.Interval)
		}
		if f.Safety.StaleAfter != nil {
			cfg.Safety.StaleAfter = time.Duration(*f.Safety.StaleAfter)
		}
	}

	if f.Scenario != nil {
		if f.Scenario.Path != nil {
			cfg.ScenarioPath = *f.Scenario.Path
		}
		if f.Scenario.Speed != nil {
			cfg.ScenarioSpeed = *f.Scenario.Speed
		}
		if f.Scenario.Loop != nil {
			cfg.ScenarioLoop = *f.Scenario.Loop
		}
	}

	if f.Storage != nil {
		if f.Storage.Path != nil {
			cfg.StoragePath = *f.Storage.Path
		}
		if f.Storage.BusyTimeout != nil {
			cfg.Storage.BusyTimeout = time.Duration(*f.Storage.BusyTimeout)
		}
		if f.Storage.Retention != nil {
			cfg.Storage.Retention = time.Duration(*f.Storage.Retention)
		}
	}

	if f.ControlPlane != nil {
		if f.ControlPlane.Listen != nil {
			cfg.ControlPlane.Listen = *f.ControlPlane.Listen
		}
		if f.ControlPlane.AllowedCIDRs != nil {
			cfg.ControlPlane.AllowedCIDRs = f.ControlPlane.AllowedCIDRs
		}
	}

	if f.Thresholds != nil {
		if f.Thresholds.TempLow != nil {
			cfg.Thresholds.TempLow = *f.Thresholds.TempLow
		}
		if f.Thresholds.TempLowHysteresis != nil {
			cfg.Thresholds.TempLowHysteresis = *f.Thresholds.TempLowHysteresis
		}
		if f.Thresholds.TempCriticalLow != nil {
			cfg.Thresholds.TempCriticalLow = *f.Thresholds.TempCriticalLow
		}
		if f.Thresholds.TempCriticalHigh != nil {
			cfg.Thresholds.TempCriticalHigh = *f.Thresholds.TempCriticalHigh
		}
		if f.Thresholds.O2Low != nil {
			cfg.Thresholds.O2Low = *f.Thresholds.O2Low
		}
		if f.Thresholds.O2LowHysteresis != nil {
			cfg.Thresholds.O2LowHysteresis = *f.Thresholds.O2LowHysteresis
		}
		if f.Thresholds.FlowMin != nil {
			cfg.Thresholds.FlowMin = *f.Thresholds.FlowMin
		}
	}
}
