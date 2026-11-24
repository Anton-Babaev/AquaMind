package policy

import (
	"fmt"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// AlarmPolicy — правило формирования алертов.
// Возвращает список алертов, которые должны быть активны для данного показания.
// Жизненный цикл (поднять один раз / снять при норме) ведёт AlertTracker,
// поэтому здесь мы лишь сообщаем, какие условия сейчас выполнены.
//
// Порядок приоритетов (верхнее главнее), см. §5.6:
//  1. Safe Mode (данные устарели / ошибка датчика) → алерт
//  2. Нет потока при включённом насосе            → алерт
//  3. Критически высокая температура              → алерт
//  4. Критически низкая температура               → алерт
//  5. Мало кислорода                              → алерт
//  6. Много кислорода                             → алерт (warning)
//  7. pH ниже нормы                               → алерт (warning)
//  8. pH выше нормы                               → алерт (warning)
func AlarmPolicy(r domain.SensorReading, safeMode bool, t domain.Thresholds) []domain.Alert {
	var alerts []domain.Alert

	// 1. Safe Mode.
	if safeMode {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertSafeMode,
			Severity: domain.SeverityCritical,
			Message:  "Данные устарели или ошибка датчика — Safe Mode",
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 2. Нет потока.
	if r.FlowLPM < t.FlowMin {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertNoFlow,
			Severity: domain.SeverityCritical,
			Message:  fmt.Sprintf("Нет потока: %.1f л/мин (мин %.1f)", r.FlowLPM, t.FlowMin),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 3. Критически высокая температура.
	if r.TempC >= t.TempCriticalHigh {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertTempCriticalHigh,
			Severity: domain.SeverityCritical,
			Message:  fmt.Sprintf("Критически высокая температура: %.1f °C", r.TempC),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 4. Критически низкая температура.
	if r.TempC <= t.TempCriticalLow {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertTempCriticalLow,
			Severity: domain.SeverityCritical,
			Message:  fmt.Sprintf("Критически низкая температура: %.1f °C", r.TempC),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 5. Мало кислорода.
	if r.O2MgL < t.O2Low {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertO2Low,
			Severity: domain.SeverityCritical,
			Message:  fmt.Sprintf("Кислород ниже порога: %.1f мг/л (мин %.1f)", r.O2MgL, t.O2Low),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 6. Много кислорода.
	if r.O2MgL > t.O2High {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertO2High,
			Severity: domain.SeverityWarning,
			Message:  fmt.Sprintf("Кислород выше порога: %.1f мг/л (макс %.1f)", r.O2MgL, t.O2High),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 7. pH ниже нормы.
	if r.PH < t.PHLow {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertPHLow,
			Severity: domain.SeverityWarning,
			Message:  fmt.Sprintf("pH ниже нормы: %.2f (мин %.2f)", r.PH, t.PHLow),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	// 8. pH выше нормы.
	if r.PH > t.PHHigh {
		alerts = append(alerts, domain.Alert{
			Type:     domain.AlertPHHigh,
			Severity: domain.SeverityWarning,
			Message:  fmt.Sprintf("pH выше нормы: %.2f (макс %.2f)", r.PH, t.PHHigh),
			At:       r.At,
			State:    domain.AlertRaised,
		})
	}

	return alerts
}
