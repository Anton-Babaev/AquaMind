package policy

import "github.com/Anton-Babaev/AquaMind/edge/domain"

// Decision — результат применения политик к одному показанию.
// Это «решение», которое ControlUsecase превращает в ActuatorState и алерты.
type Decision struct {
	Heating bool          // включать ли подогрев
	Alerts  []domain.Alert // активные алерты (поднятые в этом тике)
}

// HeatingPolicy — правило управления подогревом с гистерезисом.
//
// Порядок приоритетов (верхнее главнее), см. §5.6:
//  1. Safe Mode (данные устарели / ошибка датчика) → подогрев выкл
//  2. Нет потока при включённом насосе          → подогрев выкл (защита от перегрева)
//  3. Критически высокая температура            → подогрев выкл
//  4. Критически низкая температура             → подогрев вкл
//  5. Мало кислорода                            → на подогрев не влияет
//  6. Низкая температура                        → вкл ниже порога, выкл выше «порог + гистерезис»
//
// prevHeating — текущее состояние подогрева (для гистерезиса).
// safeMode — признак безопасного режима (данные устарели / ошибка датчика).
func HeatingPolicy(r domain.SensorReading, prevHeating bool, safeMode bool, t domain.Thresholds) bool {
	// 1. Safe Mode — подогрев выключен.
	if safeMode {
		return false
	}

	// 2. Нет потока при включённом насосе — защита от перегрева без циркуляции.
	if r.FlowLPM < t.FlowMin {
		return false
	}

	// 3. Критически высокая температура — подогрев выключен.
	if r.TempC >= t.TempCriticalHigh {
		return false
	}

	// 4. Критически низкая температура — подогрев включён.
	if r.TempC <= t.TempCriticalLow {
		return true
	}

	// 5. Мало кислорода — на подогрев не влияет (обрабатывается AlarmPolicy).

	// 6. Низкая температура с гистерезисом.
	if prevHeating {
		// Уже греем: выключаем только когда поднялись выше «порог + гистерезис».
		return r.TempC < t.TempLow+t.TempLowHysteresis
	}
	// Не греем: включаем, когда опустились ниже порога.
	return r.TempC < t.TempLow
}
