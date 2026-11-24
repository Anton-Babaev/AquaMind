package domain

import "time"

// AlertType — тип алерта.
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
	AlertRaised  AlertState = "raised"  // алерт поднят
	AlertCleared AlertState = "cleared" // алерт снят (вернулись в норму)
)

// Alert — событие тревоги. Поднимается один раз при входе в состояние
// и снимается при выходе из него (защита от спама при шумных данных).
type Alert struct {
	Type     AlertType
	Severity AlertSeverity
	Message  string
	At       time.Time // время события, UTC
	State    AlertState
}
