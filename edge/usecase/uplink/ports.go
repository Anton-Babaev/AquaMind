package uplink

import (
	"context"

	"github.com/Anton-Babaev/AquaMind/contract/syncv1"
)

// OutboxEntry — запись outbox (неотправленный тик).
type OutboxEntry struct {
	ID      int64
	Payload string
}

// Outbox — порт доступа к очереди исходящих записей.
// Реализуется SQLite-репозиторием.
type Outbox interface {
	PendingOutbox(ctx context.Context, limit int) ([]OutboxEntry, error)
	MarkSent(ctx context.Context, ids []int64) error
}

// Backend — порт отправки пакета на Backend.
// Реализуется backendclient.
type Backend interface {
	Sync(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error)
}
