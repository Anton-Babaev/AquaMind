package domain

import "fmt"

// Thresholds — пороги и гистерезис правил. Настройки control plane.
// Значения задаются в configs/edge.yaml и меняются через SettingsUsecase.
type Thresholds struct {
	TempLow           float64 `json:"temp_low"`            // нижний порог подогрева
	TempLowHysteresis float64 `json:"temp_low_hysteresis"` // гистерезис выключения подогрева
	TempCriticalLow   float64 `json:"temp_critical_low"`   // критически низкая температура
	TempCriticalHigh  float64 `json:"temp_critical_high"`  // критически высокая температура
	O2Low             float64 `json:"o2_low"`              // нижний порог кислорода
	O2High            float64 `json:"o2_high"`             // верхний порог кислорода
	O2LowHysteresis   float64 `json:"o2_low_hysteresis"`   // гистерезис снятия алерта по кислороду
	FlowMin           float64 `json:"flow_min"`            // минимальный допустимый поток
	PHLow             float64 `json:"ph_low"`              // нижняя граница pH
	PHHigh            float64 `json:"ph_high"`             // верхняя граница pH
}

// Validate проверяет корректность порогов.
// Сообщения об ошибках — человекочитаемые (их видит пользователь в UI),
// без имён полей кода и технических деталей.
func (t Thresholds) Validate() error {
	// Температура: критически низкая < нижний порог подогрева < критически высокая.
	if t.TempCriticalLow >= t.TempLow {
		return fmt.Errorf(
			"критически низкая температура (%.1f °C) должна быть ниже нижнего порога подогрева (%.1f °C)",
			t.TempCriticalLow, t.TempLow)
	}
	if t.TempLow >= t.TempCriticalHigh {
		return fmt.Errorf(
			"нижний порог подогрева (%.1f °C) должен быть ниже критически высокой температуры (%.1f °C)",
			t.TempLow, t.TempCriticalHigh)
	}
	if t.TempLowHysteresis < 0 {
		return fmt.Errorf("гистерезис подогрева не может быть отрицательным")
	}

	// Кислород: нижний порог < верхний порог.
	if t.O2Low >= t.O2High {
		return fmt.Errorf(
			"нижний порог кислорода (%.1f мг/л) должен быть ниже верхнего (%.1f мг/л)",
			t.O2Low, t.O2High)
	}
	if t.O2LowHysteresis < 0 {
		return fmt.Errorf("гистерезис кислорода не может быть отрицательным")
	}

	// Поток.
	if t.FlowMin < 0 {
		return fmt.Errorf("минимальный поток не может быть отрицательным")
	}

	// pH: нижняя граница < верхняя, обе в диапазоне 0–14.
	if t.PHLow < 0 || t.PHHigh > 14 {
		return fmt.Errorf("границы pH должны быть в диапазоне от 0 до 14")
	}
	if t.PHLow >= t.PHHigh {
		return fmt.Errorf(
			"нижняя граница pH (%.1f) должна быть ниже верхней (%.1f)",
			t.PHLow, t.PHHigh)
	}

	return nil
}
