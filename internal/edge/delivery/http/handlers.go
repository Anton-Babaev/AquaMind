package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// StateProvider — узкий интерфейс источника данных для API.
// Реализуется ControlUsecase (и, при необходимости, обёрткой с порогами).
type StateProvider interface {
	State() domain.ActuatorState
	LastReading() (domain.SensorReading, bool)
	LastAlerts() []domain.Alert
}

// ThresholdsProvider — источник и приёмник порогов (для вкладки «Пороги»).
// Реализуется хранилищем (SQLite) или usecase настроек.
type ThresholdsProvider interface {
	LoadThresholds(ctx context.Context) (domain.Thresholds, error)
	SaveThresholds(ctx context.Context, t domain.Thresholds) error
}

// AlertHistoryProvider — источник истории алертов (raised/cleared).
// Реализуется хранилищем (SQLite).
type AlertHistoryProvider interface {
	RecentAlerts(ctx context.Context, limit int) ([]domain.Alert, error)
}

// handlers — набор HTTP-обработчиков.
type handlers struct {
	state StateProvider
	// thresholds может быть nil, если вкладка порогов ещё не подключена.
	thresholds ThresholdsProvider
	// history может быть nil, если история алертов ещё не подключена.
	history AlertHistoryProvider
}

// --- DTO для JSON (отдельно от домена, чтобы контролировать формат) ---

// stateDTO — ответ GET /api/state.
type stateDTO struct {
	Reading   *readingDTO  `json:"reading"`   // null, если данных ещё нет
	Actuators actuatorsDTO `json:"actuators"` // состояние выходов
	Alerts    []alertDTO   `json:"alerts"`    // активные алерты
	At        time.Time    `json:"at"`        // время ответа сервера
}

type readingDTO struct {
	TempC   float64   `json:"temp_c"`
	FlowLPM float64   `json:"flow_lpm"`
	O2MgL   float64   `json:"o2_mg_l"`
	PH      float64   `json:"ph"`
	At      time.Time `json:"at"`
}

type actuatorsDTO struct {
	Heating  bool `json:"heating"`
	Alarm    bool `json:"alarm"`
	Normal   bool `json:"normal"`
	SafeMode bool `json:"safe_mode"`
}

type alertDTO struct {
	Type     string    `json:"type"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
	State    string    `json:"state,omitempty"` // raised/cleared (для истории)
}

// getState — GET /api/state: текущее состояние для дашборда.
func (h *handlers) getState(w http.ResponseWriter, r *http.Request) {
	reading, ok := h.state.LastReading()
	st := h.state.State()
	alerts := h.state.LastAlerts()

	resp := stateDTO{
		Actuators: actuatorsDTO{
			Heating:  st.Heating,
			Alarm:    st.Alarm,
			Normal:   st.Normal,
			SafeMode: st.SafeMode,
		},
		Alerts: make([]alertDTO, 0, len(alerts)),
		At:     time.Now().UTC(),
	}
	if ok {
		resp.Reading = &readingDTO{
			TempC:   reading.TempC,
			FlowLPM: reading.FlowLPM,
			O2MgL:   reading.O2MgL,
			PH:      reading.PH,
			At:      reading.At,
		}
	}
	for _, a := range alerts {
		resp.Alerts = append(resp.Alerts, alertDTO{
			Type:     string(a.Type),
			Severity: string(a.Severity),
			Message:  a.Message,
			At:       a.At,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// getAlerts — GET /api/alerts?limit=10: история событий алертов
// (raised/cleared), свежие первыми.
func (h *handlers) getAlerts(w http.ResponseWriter, r *http.Request) {
	if h.history == nil {
		writeError(w, http.StatusNotImplemented, "alert history not available")
		return
	}
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	alerts, err := h.history.RecentAlerts(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]alertDTO, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, alertDTO{
			Type:     string(a.Type),
			Severity: string(a.Severity),
			Message:  a.Message,
			At:       a.At,
			State:    string(a.State),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// getThresholds — GET /api/thresholds: текущие пороги.
func (h *handlers) getThresholds(w http.ResponseWriter, r *http.Request) {
	if h.thresholds == nil {
		writeError(w, http.StatusNotImplemented, "thresholds not available")
		return
	}
	t, err := h.thresholds.LoadThresholds(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// postThresholds — POST /api/thresholds: сохранить пороги.
func (h *handlers) postThresholds(w http.ResponseWriter, r *http.Request) {
	if h.thresholds == nil {
		writeError(w, http.StatusNotImplemented, "thresholds not available")
		return
	}
	var t domain.Thresholds
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := t.Validate(); err != nil {
		// Сообщение валидации уже человекочитаемое — отдаём как есть.
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.thresholds.SaveThresholds(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// --- Вспомогательные функции ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
