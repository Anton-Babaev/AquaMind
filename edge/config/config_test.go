package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// writeTempConfig создаёт временный YAML-файл с заданным содержимым
// и возвращает его путь. Файл автоматически удаляется по окончании теста.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "edge.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// TestLoad_Defaults — без файла и без env возвращаются значения по умолчанию.
func TestLoad_Defaults(t *testing.T) {
	// На всякий случай чистим env, чтобы не влиять на результат.
	t.Setenv("AQUAMIND_HARDWARE", "")
	t.Setenv("AQUAMIND_SCENARIO", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	def := Default()
	// Config содержит слайс (AllowedCIDRs), поэтому сравниваем через DeepEqual.
	if !reflect.DeepEqual(cfg, def) {
		t.Errorf("Load(\"\") = %+v, want Default() = %+v", cfg, def)
	}
}

// TestLoad_FromFile — значения из YAML-файла накладываются поверх дефолтов.
func TestLoad_FromFile(t *testing.T) {
	path := writeTempConfig(t, `
client_id: edge-test
hardware: gpio
gpio:
  heating: 5
  alarm: 6
  normal: 13
backend:
  url: "http://example.com:9000"
  sync_interval: 30s
  request_timeout: 3s
  batch_size: 42
polling:
  interval: 500ms
scenario:
  path: "scenarios/custom.csv"
  speed: 120
  loop: false
thresholds:
  temp_low: 20.0
  temp_low_hysteresis: 2.0
  temp_critical_low: 15.0
  temp_critical_high: 28.0
  o2_low: 4.0
  o2_low_hysteresis: 0.3
  flow_min: 3.0
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Проверяем, что поля из файла применились.
	if cfg.ClientID != "edge-test" {
		t.Errorf("ClientID = %q, want edge-test", cfg.ClientID)
	}
	if cfg.Hardware != HardwareGPIO {
		t.Errorf("Hardware = %q, want gpio", cfg.Hardware)
	}
	if cfg.GPIOPins != (GPIOPins{Heating: 5, Alarm: 6, Normal: 13}) {
		t.Errorf("GPIOPins = %+v", cfg.GPIOPins)
	}
	if cfg.BackendURL != "http://example.com:9000" {
		t.Errorf("BackendURL = %q", cfg.BackendURL)
	}
	if cfg.SyncInterval != 30*time.Second {
		t.Errorf("SyncInterval = %v, want 30s", cfg.SyncInterval)
	}
	if cfg.RequestTimeout != 3*time.Second {
		t.Errorf("RequestTimeout = %v, want 3s", cfg.RequestTimeout)
	}
	if cfg.BatchSize != 42 {
		t.Errorf("BatchSize = %d, want 42", cfg.BatchSize)
	}
	if cfg.PollInterval != 500*time.Millisecond {
		t.Errorf("PollInterval = %v, want 500ms", cfg.PollInterval)
	}
	if cfg.ScenarioPath != "scenarios/custom.csv" {
		t.Errorf("ScenarioPath = %q", cfg.ScenarioPath)
	}
	if cfg.ScenarioSpeed != 120 {
		t.Errorf("ScenarioSpeed = %v, want 120", cfg.ScenarioSpeed)
	}
	if cfg.ScenarioLoop {
		t.Errorf("ScenarioLoop = true, want false")
	}
	if cfg.Thresholds.TempLow != 20.0 || cfg.Thresholds.FlowMin != 3.0 {
		t.Errorf("Thresholds = %+v", cfg.Thresholds)
	}
}

// TestLoad_PartialFile — поля, отсутствующие в файле, остаются дефолтными.
func TestLoad_PartialFile(t *testing.T) {
	path := writeTempConfig(t, `
client_id: edge-partial
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	def := Default()
	if cfg.ClientID != "edge-partial" {
		t.Errorf("ClientID = %q, want edge-partial", cfg.ClientID)
	}
	// Остальное должно остаться дефолтным.
	if cfg.Hardware != def.Hardware {
		t.Errorf("Hardware = %q, want default %q", cfg.Hardware, def.Hardware)
	}
	if cfg.PollInterval != def.PollInterval {
		t.Errorf("PollInterval = %v, want default %v", cfg.PollInterval, def.PollInterval)
	}
	if cfg.Thresholds != def.Thresholds {
		t.Errorf("Thresholds = %+v, want default %+v", cfg.Thresholds, def.Thresholds)
	}
}

// TestLoad_EnvOverridesFile — переменные окружения важнее файла.
func TestLoad_EnvOverridesFile(t *testing.T) {
	path := writeTempConfig(t, `
hardware: console
scenario:
  path: "scenarios/from_file.csv"
`)

	t.Setenv("AQUAMIND_HARDWARE", "gpio")
	t.Setenv("AQUAMIND_SCENARIO", "scenarios/from_env.csv")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Hardware != HardwareGPIO {
		t.Errorf("Hardware = %q, want gpio (env overrides file)", cfg.Hardware)
	}
	if cfg.ScenarioPath != "scenarios/from_env.csv" {
		t.Errorf("ScenarioPath = %q, want from_env (env overrides file)", cfg.ScenarioPath)
	}
}

// TestLoad_RealConfigFile — читаем настоящий configs/edge.yaml из репозитория.
// Гарантирует, что реальный файл валиден и парсится без ошибок.
func TestLoad_RealConfigFile(t *testing.T) {
	// Путь относительно пакета: internal/edge/config → ../../../configs/edge.yaml
	path := filepath.Join("..", "..", "..", "configs", "edge.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("real config not found (%v), skipping", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load real config: %v", err)
	}
	// Базовые проверки, что файл реально прочитан.
	if cfg.ClientID == "" {
		t.Error("ClientID is empty")
	}
	if cfg.Hardware != HardwareConsole && cfg.Hardware != HardwareGPIO {
		t.Errorf("Hardware = %q, want console or gpio", cfg.Hardware)
	}
}

// TestLoad_Errors — набор ошибочных конфигураций.
func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid yaml",
			content: "client_id: [unclosed",
		},
		{
			name:    "invalid hardware",
			content: "hardware: usb",
		},
		{
			name:    "invalid duration",
			content: "polling:\n  interval: not-a-duration",
		},
		{
			name: "invalid thresholds",
			content: `
thresholds:
  temp_low: 10.0
  temp_critical_low: 20.0
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.content)
			if _, err := Load(path); err == nil {
				t.Errorf("Load(%q) = nil error, want error", tt.name)
			}
		})
	}
}

// TestLoad_MissingFile — несуществующий файл даёт ошибку.
func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Error("Load(missing) = nil error, want error")
	}
}

// TestLoadFromArgs — разбор флага --config.
func TestLoadFromArgs(t *testing.T) {
	path := writeTempConfig(t, "client_id: from-args\n")

	cfg, err := LoadFromArgs([]string{"--config", path})
	if err != nil {
		t.Fatalf("LoadFromArgs: %v", err)
	}
	if cfg.ClientID != "from-args" {
		t.Errorf("ClientID = %q, want from-args", cfg.ClientID)
	}
}

// TestLoadFromArgs_NoFlag — без флага работает как Load("") (дефолты).
func TestLoadFromArgs_NoFlag(t *testing.T) {
	t.Setenv("AQUAMIND_HARDWARE", "")

	cfg, err := LoadFromArgs(nil)
	if err != nil {
		t.Fatalf("LoadFromArgs: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("LoadFromArgs(nil) = %+v, want Default()", cfg)
	}
}
