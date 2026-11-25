// Package config отвечает за загрузку конфигурации Backend-сервиса.
//
// Источники настроек (в порядке возрастания приоритета):
//  1. Значения по умолчанию (функция Default) — всегда есть, даже без файла.
//  2. YAML-файл (configs/backend.yaml) — накладывается поверх дефолтов.
//  3. Переменные окружения AQUAMIND_BACKEND_* — переопределяют всё.
//
// Секреты (токен Telegram, DSN базы) на хостинге задаются ТОЛЬКО через
// переменные окружения, чтобы не попадать в git.
package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config — итоговая конфигурация Backend.
type Config struct {
	// Listen — адрес HTTP-сервера, например ":8080".
	Listen string

	// Postgres — параметры подключения к базе данных.
	Postgres PostgresConfig

	// Telegram — параметры отправки сообщений в Telegram.
	Telegram TelegramConfig

	// Notify — параметры логики уведомлений (антиспам, cooldown).
	Notify NotifyConfig

	// ReadTimeout / WriteTimeout / ShutdownTimeout — таймауты HTTP-сервера.
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// PostgresConfig — параметры подключения к PostgreSQL.
type PostgresConfig struct {
	DSN            string        // строка подключения
	MaxConns       int           // максимум соединений в пуле
	ConnectTimeout time.Duration // таймаут установки соединения
}

// TelegramConfig — параметры Telegram-бота.
type TelegramConfig struct {
	Token   string        // токен бота от @BotFather (секрет!)
	ChatID  string        // чат/канал фермера, куда слать алерты
	Timeout time.Duration // таймаут HTTP-запроса к api.telegram.org
	Enabled bool          // включена ли отправка (false → только лог)
}

// NotifyConfig — параметры логики уведомлений.
type NotifyConfig struct {
	// Cooldown — минимальный интервал между повторными уведомлениями об
	// одном и том же алерте (защита от спама при «дребезге» датчика).
	Cooldown time.Duration
	// NotifyCleared — отправлять ли сообщение о восстановлении (cleared).
	NotifyCleared bool
}

// Default возвращает конфигурацию по умолчанию.
func Default() Config {
	return Config{
		Listen: ":8080",
		Postgres: PostgresConfig{
			DSN:            "postgres://aquamind:aquamind@localhost:5432/aquamind?sslmode=disable",
			MaxConns:       10,
			ConnectTimeout: 5 * time.Second,
		},
		Telegram: TelegramConfig{
			Token:   "",
			ChatID:  "",
			Timeout: 10 * time.Second,
			Enabled: false,
		},
		Notify: NotifyConfig{
			Cooldown:      5 * time.Minute,
			NotifyCleared: true,
		},
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// Load собирает конфигурацию из трёх источников:
// дефолты → YAML-файл (если path != "") → переменные окружения.
func Load(path string) (Config, error) {
	cfg := Default()

	if path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return cfg, err
		}
	}

	if err := applyEnv(&cfg); err != nil {
		return cfg, err
	}

	if err := cfg.validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadFromArgs разбирает флаг --config <путь> и загружает конфигурацию.
func LoadFromArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("backend", flag.ContinueOnError)
	configPath := fs.String("config", "", "путь к YAML-файлу конфигурации (например, configs/backend.yaml)")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	return Load(*configPath)
}

// loadFile читает YAML-файл и накладывает значения на cfg.
func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}
	var raw fileConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse yaml %q: %w", path, err)
	}
	raw.applyTo(cfg)
	return nil
}

// applyEnv переопределяет поля значениями из переменных окружения.
// Секреты (токен, DSN) удобно задавать именно так на хостинге.
func applyEnv(cfg *Config) error {
	if v := os.Getenv("AQUAMIND_BACKEND_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("AQUAMIND_BACKEND_POSTGRES_DSN"); v != "" {
		cfg.Postgres.DSN = v
	}
	if v := os.Getenv("AQUAMIND_BACKEND_TG_TOKEN"); v != "" {
		cfg.Telegram.Token = v
		cfg.Telegram.Enabled = true
	}
	if v := os.Getenv("AQUAMIND_BACKEND_TG_CHAT_ID"); v != "" {
		cfg.Telegram.ChatID = v
	}
	if v := os.Getenv("AQUAMIND_BACKEND_TG_ENABLED"); v != "" {
		cfg.Telegram.Enabled = v == "1" || v == "true"
	}
	if v := os.Getenv("AQUAMIND_BACKEND_NOTIFY_COOLDOWN"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid AQUAMIND_BACKEND_NOTIFY_COOLDOWN: %w", err)
		}
		cfg.Notify.Cooldown = d
	}
	return nil
}

// validate проверяет корректность итоговой конфигурации.
func (c Config) validate() error {
	if c.Listen == "" {
		return fmt.Errorf("listen address must not be empty")
	}
	if c.Postgres.DSN == "" {
		return fmt.Errorf("postgres dsn must not be empty")
	}
	if c.Telegram.Enabled && (c.Telegram.Token == "" || c.Telegram.ChatID == "") {
		return fmt.Errorf("telegram enabled but token or chat_id is empty")
	}
	return nil
}

// ---------------------------------------------------------------------------
// «Сырые» структуры для разбора YAML (поля-указатели: nil = «не задано»).
// ---------------------------------------------------------------------------

type fileConfig struct {
	Listen   string        `yaml:"listen"`
	Postgres *filePostgres `yaml:"postgres"`
	Telegram *fileTelegram `yaml:"telegram"`
	Notify   *fileNotify   `yaml:"notify"`
}

type filePostgres struct {
	DSN            *string   `yaml:"dsn"`
	MaxConns       *int      `yaml:"max_conns"`
	ConnectTimeout *duration `yaml:"connect_timeout"`
}

type fileTelegram struct {
	Token   *string   `yaml:"token"`
	ChatID  *string   `yaml:"chat_id"`
	Timeout *duration `yaml:"timeout"`
	Enabled *bool     `yaml:"enabled"`
}

type fileNotify struct {
	Cooldown      *duration `yaml:"cooldown"`
	NotifyCleared *bool     `yaml:"notify_cleared"`
}

// duration — обёртка над time.Duration для разбора строк "10s", "500ms".
type duration time.Duration

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

// applyTo переносит заданные (не nil) поля в итоговую Config.
func (f fileConfig) applyTo(cfg *Config) {
	if f.Listen != "" {
		cfg.Listen = f.Listen
	}
	if f.Postgres != nil {
		if f.Postgres.DSN != nil {
			cfg.Postgres.DSN = *f.Postgres.DSN
		}
		if f.Postgres.MaxConns != nil {
			cfg.Postgres.MaxConns = *f.Postgres.MaxConns
		}
		if f.Postgres.ConnectTimeout != nil {
			cfg.Postgres.ConnectTimeout = time.Duration(*f.Postgres.ConnectTimeout)
		}
	}
	if f.Telegram != nil {
		if f.Telegram.Token != nil {
			cfg.Telegram.Token = *f.Telegram.Token
		}
		if f.Telegram.ChatID != nil {
			cfg.Telegram.ChatID = *f.Telegram.ChatID
		}
		if f.Telegram.Timeout != nil {
			cfg.Telegram.Timeout = time.Duration(*f.Telegram.Timeout)
		}
		if f.Telegram.Enabled != nil {
			cfg.Telegram.Enabled = *f.Telegram.Enabled
		}
	}
	if f.Notify != nil {
		if f.Notify.Cooldown != nil {
			cfg.Notify.Cooldown = time.Duration(*f.Notify.Cooldown)
		}
		if f.Notify.NotifyCleared != nil {
			cfg.Notify.NotifyCleared = *f.Notify.NotifyCleared
		}
	}
}
