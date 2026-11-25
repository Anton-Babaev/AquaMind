// Package domain содержит сущности Backend-сервиса.
//
// Backend принимает данные от Edge (показания, состояние выходов, алерты),
// хранит их в PostgreSQL и уведомляет фермера через сторонние сервисы
// (Telegram, позже SMS).
package domain

import "time"

// Reading — показание датчиков.
type Reading struct {
	TempC   float64
	FlowLPM float64
	O2MgL   float64
	PH      float64
	At      time.Time // время измерения, UTC
}

// ActuatorState — состояние выходов Edge в момент тика.
type ActuatorState struct {
	Heating  bool
	Alarm    bool
	Normal   bool
	SafeMode bool
}

// AlertType — тип алерта (совпадает с Edge).
type AlertType string

const (
	AlertO2Low            AlertType = "o2_low"
	AlertO2High           AlertType = "o2_high"
	AlertNoFlow           AlertType = "no_flow"
	AlertTempCriticalLow  AlertType = "temp_critical_low"
	AlertTempCriticalHigh AlertType = "temp_critical_high"
	AlertPHLow            AlertType = "ph_low"
	AlertPHHigh           AlertType = "ph_high"
	AlertSafeMode         AlertType = "safe_mode"
)

// AlertSeverity — уровень важности алерта.
type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// AlertState — состояние жизненного цикла алерта.
type AlertState string

const (
	AlertRaised  AlertState = "raised"
	AlertCleared AlertState = "cleared"
)

// Alert — событие тревоги.
type Alert struct {
	Type     AlertType
	Severity AlertSeverity
	Message  string
	At       time.Time
	State    AlertState
}

// Tick — один тик от Edge: показание + состояние выходов + алерты.
type Tick struct {
	Reading  Reading
	Actuator ActuatorState
	Alerts   []Alert
}

// Client — известное устройство Edge.
type Client struct {
	ClientID  string
	FirstSeen time.Time
	LastSeen  time.Time
}
