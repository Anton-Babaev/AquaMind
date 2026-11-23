package domain

import "fmt"

// Thresholds — пороги и гистерезис правил. Настройки control plane.
// Значения задаются в configs/edge.yaml и меняются через SettingsUsecase.
type Thresholds struct {
	TempLow           float64 // нижний порог подогрева
	TempLowHysteresis float64 // гистерезис выключения подогрева
	TempCriticalLow   float64 // критически низкая температура
	TempCriticalHigh  float64 // критически высокая температура
	O2Low             float64 // нижний порог кислорода
	O2LowHysteresis   float64 // гистерезис снятия алерта по кислороду
	FlowMin           float64 // минимальный допустимый поток
}

// Validate проверяет корректность порогов.
func (t Thresholds) Validate() error {
	if t.TempCriticalLow >= t.TempLow {
		return fmt.Errorf("temp_critical_low (%.2f) must be < temp_low (%.2f)", t.TempCriticalLow, t.TempLow)
	}
	if t.TempLow >= t.TempCriticalHigh {
		return fmt.Errorf("temp_low (%.2f) must be < temp_critical_high (%.2f)", t.TempLow, t.TempCriticalHigh)
	}
	if t.TempLowHysteresis < 0 {
		return fmt.Errorf("temp_low_hysteresis must be >= 0")
	}
	if t.O2LowHysteresis < 0 {
		return fmt.Errorf("o2_low_hysteresis must be >= 0")
	}
	if t.FlowMin < 0 {
		return fmt.Errorf("flow_min must be >= 0")
	}
	return nil
}
