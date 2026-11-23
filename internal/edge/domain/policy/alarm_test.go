package policy

import (
	"testing"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

func hasAlert(alerts []domain.Alert, typ domain.AlertType) bool {
	for _, a := range alerts {
		if a.Type == typ {
			return true
		}
	}
	return false
}

func TestAlarmPolicy(t *testing.T) {
	th := testThresholds()

	tests := []struct {
		name     string
		reading  domain.SensorReading
		safeMode bool
		want     []domain.AlertType
	}{
		{
			name:    "всё в норме — нет алертов",
			reading: domain.SensorReading{TempC: 24, FlowLPM: 12, O2MgL: 7},
			want:    nil,
		},
		{
			name:     "safe mode — алерт",
			reading:  domain.SensorReading{TempC: 24, FlowLPM: 12, O2MgL: 7},
			safeMode: true,
			want:     []domain.AlertType{domain.AlertSafeMode},
		},
		{
			name:    "нет потока — алерт",
			reading: domain.SensorReading{TempC: 24, FlowLPM: 2, O2MgL: 7},
			want:    []domain.AlertType{domain.AlertNoFlow},
		},
		{
			name:    "критически высокая t° — алерт",
			reading: domain.SensorReading{TempC: 31, FlowLPM: 12, O2MgL: 7},
			want:    []domain.AlertType{domain.AlertTempCriticalHigh},
		},
		{
			name:    "критически низкая t° — алерт",
			reading: domain.SensorReading{TempC: 17, FlowLPM: 12, O2MgL: 7},
			want:    []domain.AlertType{domain.AlertTempCriticalLow},
		},
		{
			name:    "мало кислорода — алерт",
			reading: domain.SensorReading{TempC: 24, FlowLPM: 12, O2MgL: 4},
			want:    []domain.AlertType{domain.AlertO2Low},
		},
		{
			name:    "холодно и мало кислорода — два алерта",
			reading: domain.SensorReading{TempC: 17, FlowLPM: 12, O2MgL: 4},
			want:    []domain.AlertType{domain.AlertTempCriticalLow, domain.AlertO2Low},
		},
		{
			name:     "safe mode + нет потока — два алерта",
			reading:  domain.SensorReading{TempC: 24, FlowLPM: 2, O2MgL: 7},
			safeMode: true,
			want:     []domain.AlertType{domain.AlertSafeMode, domain.AlertNoFlow},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AlarmPolicy(tt.reading, tt.safeMode, th)
			if len(got) != len(tt.want) {
				t.Fatalf("AlarmPolicy() вернул %d алертов, want %d (%v)", len(got), len(tt.want), tt.want)
			}
			for _, wantType := range tt.want {
				if !hasAlert(got, wantType) {
					t.Errorf("AlarmPolicy() не содержит алерт %q", wantType)
				}
			}
		})
	}
}
