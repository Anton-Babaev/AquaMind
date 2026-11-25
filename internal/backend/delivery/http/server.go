// Package http — HTTP-слой Backend: приём синхронизации от Edge, чтение
// данных для Frontend, health-check и раздача статики дашборда.
package http

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
)

//go:embed static
var staticFS embed.FS

// Ingest — usecase приёма пакетов (реализуется usecase/ingest).
type Ingest interface {
	Handle(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error)
}

// Pinger — проверка доступности БД (реализуется adapter/postgres).
type Pinger interface {
	Ping(ctx context.Context) error
}

// Server — HTTP-сервер Backend.
type Server struct {
	ingest Ingest
	pinger Pinger
	query  Query
}

// NewServer создаёт HTTP-сервер.
func NewServer(ingest Ingest, pinger Pinger, query Query) *Server {
	return &Server{ingest: ingest, pinger: pinger, query: query}
}

// Handler возвращает http.Handler со всеми маршрутами.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Приём данных от Edge.
	mux.HandleFunc("/api/v1/sync", s.handleSync)
	mux.HandleFunc("/healthz", s.handleHealth)

	// Чтение данных для Frontend.
	mux.HandleFunc("/api/v1/clients", s.handleClients)
	mux.HandleFunc("/api/v1/state", s.handleState)
	mux.HandleFunc("/api/v1/ticks", s.handleTicks)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)

	// Статика дашборда (index.html, app.js, style.css).
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Printf("static fs: %v", err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(static)))
	}

	return mux
}

// handleSync принимает пакет синхронизации от Edge.
func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batch syncv1.SyncBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	ack, err := s.ingest.Handle(r.Context(), batch)
	if err != nil {
		// Данные могли сохраниться, но уведомление упало — логируем, но
		// отвечаем 500, чтобы Edge повторил (дедуп защитит от дублей).
		log.Printf("sync error: client=%s ticks=%d err=%v", batch.ClientID, len(batch.Ticks), err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("sync: client=%s ticks=%d accepted=%d duplicates=%d",
		batch.ClientID, len(batch.Ticks), ack.Accepted, ack.Duplicates)

	writeJSON(w, ack)
}

// handleHealth проверяет доступность БД.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if s.pinger != nil {
		if err := s.pinger.Ping(ctx); err != nil {
			http.Error(w, "db unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleClients отдаёт список известных устройств.
func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clients, err := s.query.Clients(r.Context())
	if err != nil {
		log.Printf("query clients: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, clientsDTO(clients))
}

// handleState отдаёт последнее состояние устройства (?client_id=...).
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "missing client_id", http.StatusBadRequest)
		return
	}
	tick, ok, err := s.query.State(r.Context(), clientID)
	if err != nil {
		log.Printf("query state: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "no data", http.StatusNotFound)
		return
	}
	writeJSON(w, toTickDTO(tick))
}

// handleTicks отдаёт историю показаний (?client_id=...&from=...&to=...&limit=...).
// from/to — Unix-миллисекунды (необязательны).
func (s *Server) handleTicks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	clientID := q.Get("client_id")
	if clientID == "" {
		http.Error(w, "missing client_id", http.StatusBadRequest)
		return
	}
	from := parseMillis(q.Get("from"))
	to := parseMillis(q.Get("to"))
	limit := parseInt(q.Get("limit"), 0)

	ticks, err := s.query.Ticks(r.Context(), clientID, from, to, limit)
	if err != nil {
		log.Printf("query ticks: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, ticksDTO(ticks))
}

// handleAlerts отдаёт историю алертов (?client_id=...&limit=...).
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	clientID := q.Get("client_id")
	if clientID == "" {
		http.Error(w, "missing client_id", http.StatusBadRequest)
		return
	}
	limit := parseInt(q.Get("limit"), 0)

	alerts, err := s.query.Alerts(r.Context(), clientID, limit)
	if err != nil {
		log.Printf("query alerts: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, alertsDTO(alerts))
}

// writeJSON пишет JSON-ответ.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
	}
}

// parseMillis разбирает Unix-миллисекунды; пустая строка/ошибка → нулевое время.
func parseMillis(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// parseInt разбирает целое; пустая строка/ошибка → def.
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
