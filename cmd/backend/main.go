package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
)

// server — минимальный Backend-заглушка для проверки синхронизации.
// Дедупликация по (client_id, at) в памяти. Полноценный Backend (PostgreSQL,
// Telegram) — зона Романа, здесь только приёмник для сквозной проверки.
type server struct {
	mu   sync.Mutex
	seen map[string]struct{} // ключ: client_id|at
}

func newServer() *server {
	return &server{seen: make(map[string]struct{})}
}

func (s *server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batch syncv1.SyncBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	accepted, duplicates := 0, 0
	s.mu.Lock()
	for _, t := range batch.Ticks {
		key := batch.ClientID + "|" + t.Reading.At.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		if _, ok := s.seen[key]; ok {
			duplicates++
			continue
		}
		s.seen[key] = struct{}{}
		accepted++
	}
	s.mu.Unlock()

	log.Printf("sync: client=%s ticks=%d accepted=%d duplicates=%d",
		batch.ClientID, len(batch.Ticks), accepted, duplicates)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncv1.SyncAck{Accepted: accepted, Duplicates: duplicates})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	s := newServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sync", s.handleSync)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	addr := ":8080"
	log.Printf("backend stub listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("backend: %v", err)
	}
}
