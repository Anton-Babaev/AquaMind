package syncv1

import "time"

// Версия протокола синхронизации.
const Version = 1

// ReadingDTO — показание датчиков в пакете синхронизации.
type ReadingDTO struct {
	TempC   float64   `json:"temp_c"`
	FlowLPM float64   `json:"flow_lpm"`
	O2MgL   float64   `json:"o2_mgl"`
	PH      float64   `json:"ph"`
	At      time.Time `json:"at"`
}

// ActuatorDTO — состояние выходов в пакете синхронизации.
type ActuatorDTO struct {
	Heating  bool `json:"heating"`
	Alarm    bool `json:"alarm"`
	Normal   bool `json:"normal"`
	SafeMode bool `json:"safe_mode"`
}

// AlertDTO — алерт в пакете синхронизации.
type AlertDTO struct {
	Type     string    `json:"type"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
	State    string    `json:"state"`
}

// TickDTO — один тик: показание + состояние выходов + алерты.
type TickDTO struct {
	Reading  ReadingDTO  `json:"reading"`
	Actuator ActuatorDTO `json:"actuator"`
	Alerts   []AlertDTO  `json:"alerts,omitempty"`
}

// SyncBatch — пакет тиков от Edge к Backend.
type SyncBatch struct {
	Version  int       `json:"version"`
	ClientID string    `json:"client_id"`
	Ticks    []TickDTO `json:"ticks"`
}

// SyncAck — ответ Backend на пакет.
type SyncAck struct {
	Accepted int `json:"accepted"` // сколько тиков принято
	Duplicates int `json:"duplicates"` // сколько отброшено как дубликаты
}
