package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Anton-Babaev/AquaMind/backend/adapter/postgres"
	"github.com/Anton-Babaev/AquaMind/backend/adapter/telegram"
	"github.com/Anton-Babaev/AquaMind/backend/config"
	backendhttp "github.com/Anton-Babaev/AquaMind/backend/delivery/http"
	"github.com/Anton-Babaev/AquaMind/backend/usecase/ingest"
	"github.com/Anton-Babaev/AquaMind/backend/usecase/notify"
	"github.com/Anton-Babaev/AquaMind/backend/usecase/query"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Загружаем конфигурацию: флаг --config <путь>, иначе дефолты + env.
	cfg, err := config.LoadFromArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Сборка зависимостей (DI) ---

	// Хранилище: PostgreSQL.
	store, err := postgres.Open(ctx, cfg.Postgres.DSN, cfg.Postgres.MaxConns, cfg.Postgres.ConnectTimeout)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer store.Close()

	// Уведомления: Telegram (или лог-заглушка, если выключено).
	notifier := buildNotifier(cfg)

	// UseCase приёма.
	ing := ingest.New(store, notifier)

	// UseCase чтения (для Frontend).
	qry := query.New(store)

	// HTTP-сервер.
	srv := backendhttp.NewServer(ing, store, qry)
	httpServer := &http.Server{
		Addr:         cfg.Listen,
		Handler:      srv.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// --- Запуск ---

	go func() {
		log.Printf("backend listening on %s (telegram=%v)", cfg.Listen, cfg.Telegram.Enabled)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	// Ждём сигнал остановки.
	<-ctx.Done()
	log.Printf("backend shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	log.Printf("backend stopped")
}

// buildNotifier создаёт notifier: Telegram, если включён, иначе лог-заглушку.
func buildNotifier(cfg config.Config) *notify.Usecase {
	if cfg.Telegram.Enabled {
		sender := telegram.New(cfg.Telegram.Token, cfg.Telegram.ChatID, cfg.Telegram.Timeout)
		log.Printf("notifier: telegram (chat_id=%s)", cfg.Telegram.ChatID)
		return notify.New(sender, cfg.Notify.Cooldown, cfg.Notify.NotifyCleared)
	}
	log.Printf("notifier: log-only (telegram disabled)")
	return notify.New(logSender{}, cfg.Notify.Cooldown, cfg.Notify.NotifyCleared)
}

// logSender — заглушка notify.Sender: пишет сообщения в лог.
// Используется, когда Telegram не настроен (разработка/отладка).
type logSender struct{}

func (logSender) Name() string { return "log" }

func (logSender) Send(_ context.Context, text string) error {
	log.Printf("[NOTIFY] %s", text)
	return nil
}
