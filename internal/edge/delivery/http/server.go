// Package http реализует локальный веб-интерфейс Edge-контроллера:
// JSON API для состояния/порогов и раздачу статики (дашборд в браузере).
//
// Слой доставки: зависит от usecase (через узкие интерфейсы), но не наоборот.
// На Pi статика открывается в kiosk-браузере на мониторе; на ноутбуке —
// в обычном браузере по http://localhost:8081.
package http

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed static
var staticFS embed.FS

// Server — HTTP-сервер веб-интерфейса Edge.
type Server struct {
	httpServer *http.Server
}

// NewServer создаёт сервер, привязанный к адресу listen (например, ":8081").
// state — источник данных для дашборда (реализуется ControlUsecase).
// thresholds — источник/приёмник порогов (реализуется хранилищем); может быть nil.
// history — источник истории алертов (реализуется хранилищем); может быть nil.
func NewServer(listen string, state StateProvider, thresholds ThresholdsProvider, history AlertHistoryProvider) *Server {
	mux := http.NewServeMux()

	h := &handlers{state: state, thresholds: thresholds, history: history}

	// JSON API.
	mux.HandleFunc("GET /api/state", h.getState)
	mux.HandleFunc("GET /api/alerts", h.getAlerts)
	mux.HandleFunc("GET /api/thresholds", h.getThresholds)
	mux.HandleFunc("POST /api/thresholds", h.postThresholds)

	// Статика (дашборд). Отдаём содержимое подкаталога static/ как корень.
	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		// Не должно случаться: embed гарантирует наличие каталога.
		panic("http: embed static: " + err.Error())
	}
	mux.Handle("GET /", http.FileServer(http.FS(staticRoot)))

	return &Server{
		httpServer: &http.Server{
			Addr:              listen,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run запускает сервер и блокируется до отмены ctx, после чего выполняет
// graceful shutdown. Возвращает nil при штатной остановке.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		log.Printf("http: listening on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		// Даём активным запросам завершиться.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		log.Printf("http: stopped")
		return nil
	case err := <-errCh:
		return err
	}
}
