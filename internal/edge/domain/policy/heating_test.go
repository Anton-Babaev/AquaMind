package policy

import (
	"testing"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

func testThresholds() domain.Thresholds {
	return domain.Thresholds{
		TempLow:           22.0,
		TempLowHysteresis: 1.0,
		TempCriticalLow:   18.0,
		TempCriticalHigh:  30.0,
		O2Low:             5.0,
		O2LowHysteresis:   0.5,
		FlowMin:           5.0,
	}
}

func TestHeatingPolicy(t *testing.T) {
	th := testThresholds()

	tests := []struct {
		name        string
		reading     domain.SensorReading
		prevHeating bool
		safeMode    bool
		want        bool
	}{
		// 1. Safe Mode главнее всего.
		{"safe mode выключает подогрев даже при холоде", domain.SensorReading{TempC: 10, FlowLPM: 12}, true, true, false},

		// 2. Нет потока — подогрев выкл (защита от перегрева).
		{"нет потока выключает подогрев", domain.SensorReading{TempC: 20, FlowLPM: 2}, true, false, false},

		// 3. Критически высокая температура — выкл.
		{"критически высокая t° выключает подогрев", domain.SensorReading{TempC: 31, FlowLPM: 12}, true, false, false},

		// 4. Критически низкая температура — вкл.
		{"критически низкая t° включает подогрев", domain.SensorReading{TempC: 17, FlowLPM: 12}, false, false, true},

		// 6. Низкая температура с гистерезисом.
		{"ниже порога, не грели — включаем", domain.SensorReading{TempC: 21.5, FlowLPM: 12}, false, false, true},
		{"выше порога, не грели — не включаем", domain.SensorReading{TempC: 22.5, FlowLPM: 12}, false, false, false},
		{"грели, ещё ниже порог+гистерезис — продолжаем греть", domain.SensorReading{TempC: 22.5, FlowLPM: 12}, true, false, true},
		{"грели, дошли до порог+гистерезис — выключаем", domain.SensorReading{TempC: 23.0, FlowLPM: 12}, true, false, false},
		{"грели, выше порог+гистерезис — выключаем", domain.SensorReading{TempC: 23.5, FlowLPM: 12}, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeatingPolicy(tt.reading, tt.prevHeating, tt.safeMode, th)
			if got != tt.want {
				t.Errorf("HeatingPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}
