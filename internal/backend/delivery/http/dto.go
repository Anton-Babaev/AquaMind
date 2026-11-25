package http

import (
	"context"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/backend/domain"
)

// Query — usecase чтения данных (реализуется usecase/query).
type Query interface {
	Clients(ctx context.Context) ([]domain.Client, error)
	State(ctx context.Context, clientID string) (domain.Tick, bool, error)
	Ticks(ctx context.Context, clientID string, from, to time.Time, limit int) ([]domain.Tick, error)
	Alerts(ctx context.Context, clientID string, limit int) ([]domain.Alert, error)
}

// clientDTO — устройство для JSON-ответа.
type clientDTO struct {
	ClientID  string `json:"client_id"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

// readingDTO — показания датчиков.
type readingDTO struct {
	TempC   float64 `json:"temp_c"`
	FlowLPM float64 `json:"flow_lpm"`
	O2MgL   float64 `json:"o2_mgl"`
	PH      float64 `json:"ph"`
	At      string  `json:"at"`
}

// actuatorDTO — состояние выходов.
type actuatorDTO struct {
	Heating  bool `json:"heating"`
	Alarm    bool `json:"alarm"`
	Normal   bool `json:"normal"`
	SafeMode bool `json:"safe_mode"`
}

// tickDTO — тик для JSON-ответа.
type tickDTO struct {
	Reading  readingDTO  `json:"reading"`
	Actuator actuatorDTO `json:"actuator"`
}

// alertDTO — событие алерта для JSON-ответа.
type alertDTO struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	At       string `json:"at"`
	State    string `json:"state"`
}

// clientsDTO преобразует список клиентов.
func clientsDTO(clients []domain.Client) []clientDTO {
	out := make([]clientDTO, 0, len(clients))
	for _, c := range clients {
		out = append(out, clientDTO{
			ClientID:  c.ClientID,
			FirstSeen: c.FirstSeen.UTC().Format(time.RFC3339),
			LastSeen:  c.LastSeen.UTC().Format(time.RFC3339),
		})
	}
	return out
}

// toTickDTO преобразует один тик.
func toTickDTO(t domain.Tick) tickDTO {
	return tickDTO{
		Reading: readingDTO{
			TempC:   t.Reading.TempC,
			FlowLPM: t.Reading.FlowLPM,
			O2MgL:   t.Reading.O2MgL,
			PH:      t.Reading.PH,
			At:      t.Reading.At.UTC().Format(time.RFC3339),
		},
		Actuator: actuatorDTO{
			Heating:  t.Actuator.Heating,
			Alarm:    t.Actuator.Alarm,
			Normal:   t.Actuator.Normal,
			SafeMode: t.Actuator.SafeMode,
		},
	}
}

// ticksDTO преобразует список тиков.
func ticksDTO(ticks []domain.Tick) []tickDTO {
	out := make([]tickDTO, 0, len(ticks))
	for _, t := range ticks {
		out = append(out, toTickDTO(t))
	}
	return out
}

// alertsDTO преобразует список алертов.

func alertsDTO(alerts []domain.Alert) []alertDTO {
	out := make([]alertDTO, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, alertDTO{
			Type:     string(a.Type),
			Severity: string(a.Severity),
			Message:  a.Message,
			At:       a.At.UTC().Format(time.RFC3339),
			State:    string(a.State),
		})
	}
	return out
}
