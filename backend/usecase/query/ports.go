package query

import (
	"context"
	"time"

	"github.com/Anton-Babaev/AquaMind/backend/domain"
)

// Reader — хранилище для чтения (реализуется adapter/postgres).
type Reader interface {
	// ListClients возвращает всех известных клиентов.
	ListClients(ctx context.Context) ([]domain.Client, error)
	// LatestTick возвращает последний тик клиента (ok=false, если нет).
	LatestTick(ctx context.Context, clientID string) (domain.Tick, bool, error)
	// TicksRange возвращает тики клиента в диапазоне [from, to], свежие первыми.
	TicksRange(ctx context.Context, clientID string, from, to time.Time, limit int) ([]domain.Tick, error)
	// Alerts возвращает последние события алертов клиента.
	Alerts(ctx context.Context, clientID string, limit int) ([]domain.Alert, error)
}
