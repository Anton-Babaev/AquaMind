package sqlite

import (
	"strconv"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// Ключи настроек порогов.
const (
	keyTempLow           = "temp_low"
	keyTempLowHysteresis = "temp_low_hysteresis"
	keyTempCriticalLow   = "temp_critical_low"
	keyTempCriticalHigh  = "temp_critical_high"
	keyO2Low             = "o2_low"
	keyO2High            = "o2_high"
	keyO2LowHysteresis   = "o2_low_hysteresis"
	keyFlowMin           = "flow_min"
	keyPHLow             = "ph_low"
	keyPHHigh            = "ph_high"
)

// thresholdsToMap превращает пороги в map для settings.
func thresholdsToMap(t domain.Thresholds) map[string]string {
	return map[string]string{
		keyTempLow:           strconv.FormatFloat(t.TempLow, 'f', -1, 64),
		keyTempLowHysteresis: strconv.FormatFloat(t.TempLowHysteresis, 'f', -1, 64),
		keyTempCriticalLow:   strconv.FormatFloat(t.TempCriticalLow, 'f', -1, 64),
		keyTempCriticalHigh:  strconv.FormatFloat(t.TempCriticalHigh, 'f', -1, 64),
		keyO2Low:             strconv.FormatFloat(t.O2Low, 'f', -1, 64),
		keyO2High:            strconv.FormatFloat(t.O2High, 'f', -1, 64),
		keyO2LowHysteresis:   strconv.FormatFloat(t.O2LowHysteresis, 'f', -1, 64),
		keyFlowMin:           strconv.FormatFloat(t.FlowMin, 'f', -1, 64),
		keyPHLow:             strconv.FormatFloat(t.PHLow, 'f', -1, 64),
		keyPHHigh:            strconv.FormatFloat(t.PHHigh, 'f', -1, 64),
	}
}

// thresholdsFromMap собирает пороги из map. Отсутствующие ключи берутся
// из значений по умолчанию.
func thresholdsFromMap(m map[string]string) domain.Thresholds {
	def := domain.Thresholds{
		TempLow:           22.0,
		TempLowHysteresis: 1.0,
		TempCriticalLow:   18.0,
		TempCriticalHigh:  30.0,
		O2Low:             5.0,
		O2High:            9.0,
		O2LowHysteresis:   0.5,
		FlowMin:           5.0,
		PHLow:             6.5,
		PHHigh:            8.5,
	}
	get := func(key string, fallback float64) float64 {
		if v, ok := m[key]; ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
		return fallback
	}
	return domain.Thresholds{
		TempLow:           get(keyTempLow, def.TempLow),
		TempLowHysteresis: get(keyTempLowHysteresis, def.TempLowHysteresis),
		TempCriticalLow:   get(keyTempCriticalLow, def.TempCriticalLow),
		TempCriticalHigh:  get(keyTempCriticalHigh, def.TempCriticalHigh),
		O2Low:             get(keyO2Low, def.O2Low),
		O2High:            get(keyO2High, def.O2High),
		O2LowHysteresis:   get(keyO2LowHysteresis, def.O2LowHysteresis),
		FlowMin:           get(keyFlowMin, def.FlowMin),
		PHLow:             get(keyPHLow, def.PHLow),
		PHHigh:            get(keyPHHigh, def.PHHigh),
	}
}
