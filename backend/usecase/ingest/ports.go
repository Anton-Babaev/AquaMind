package ingest

import (
	"context"

	"github.com/Anton-Babaev/AquaMind/backend/domain"
)

// Store — хранилище тиков (реализуется adapter/postgres).
type Store interface {
	// SaveBatch сохраняет пакет тиков клиента, возвращает (принято, дубликатов).
	SaveBatch(ctx context.Context, clientID string, ticks []domain.Tick) (accepted, duplicates int, err error)
}

// Notifier — отправка уведомлений о новых алертах (реализуется adapter/telegram).
type Notifier interface {
	// Notify обрабатывает алерты из принятых тиков и решает, что отправить.
	Notify(ctx context.Context, clientID string, alerts []domain.Alert) error
}
