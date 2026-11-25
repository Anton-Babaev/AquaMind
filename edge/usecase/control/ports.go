package control

import (
	"context"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// SensorSource — порт чтения показаний (HAL, часть 1).
// Реализации: SimulationProvider (MVP), в будущем ModbusScanner.
type SensorSource interface {
	Read(ctx context.Context) (domain.SensorReading, error)
}

// Actuators — порт управления выходами (HAL, часть 2).
// Реализации: GPIOLEDProvider (Pi), console (заглушка для разработки).
type Actuators interface {
	Apply(ctx context.Context, s domain.ActuatorState) error
}

// Store — порт хранения. Показание, состояние выходов, алерты и записи outbox
// сохраняются одной транзакцией (правило 12).
type Store interface {
	SaveTick(ctx context.Context, r domain.SensorReading, s domain.ActuatorState, alerts []domain.Alert) error
	LoadThresholds(ctx context.Context) (domain.Thresholds, error)
}

// AlertTracker — порт жизненного цикла алертов.
// Поднимает алерт один раз при входе в состояние и снимает при выходе.
type AlertTracker interface {
	// Track принимает активные алерты текущего тика и возвращает
	// события, которые нужно сохранить (raised/cleared).
	Track(active []domain.Alert) []domain.Alert

	// IsActive сообщает, активен ли сейчас алерт данного типа.
	// Нужен для гистерезиса (например, кислород держим до «порог + гистерезис»).
	IsActive(typ domain.AlertType) bool
}
