package syncworker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
	"github.com/Anton-Babaev/AquaMind/internal/edge/adapter/storage/sqlite"
)

// OutboxStore — порт доступа к outbox.
type OutboxStore interface {
	PendingOutbox(ctx context.Context, limit int) ([]sqlite.OutboxEntry, error)
	MarkSent(ctx context.Context, ids []int64) error
}

// Backend — порт отправки на Backend.
type Backend interface {
	Sync(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error)
}

// Worker — delivery-компонент: периодически забирает outbox и отправляет батчами.
type Worker struct {
	store    OutboxStore
	backend  Backend
	clientID string
	interval time.Duration
	batch    int
}

// New создаёт SyncWorker.
func New(store OutboxStore, backend Backend, clientID string, interval time.Duration, batch int) *Worker {
	if batch <= 0 {
		batch = 100
	}
	return &Worker{store: store, backend: backend, clientID: clientID, interval: interval, batch: batch}
}

// Run запускает цикл синхронизации до отмены контекста.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.syncOnce(ctx); err != nil {
				log.Printf("syncworker: %v", err)
			}
		}
	}
}

// syncOnce выполняет одну попытку отправки.
func (w *Worker) syncOnce(ctx context.Context) error {
	entries, err := w.store.PendingOutbox(ctx, w.batch)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	ticks := make([]syncv1.TickDTO, 0, len(entries))
	ids := make([]int64, 0, len(entries))
	for _, e := range entries {
		var t syncv1.TickDTO
		if err := json.Unmarshal([]byte(e.Payload), &t); err != nil {
			log.Printf("syncworker: skip bad payload id=%d: %v", e.ID, err)
			// Помечаем как отправленный, чтобы не зацикливаться на битой записи.
			ids = append(ids, e.ID)
			continue
		}
		ticks = append(ticks, t)
		ids = append(ids, e.ID)
	}

	if len(ticks) == 0 {
		return w.store.MarkSent(ctx, ids)
	}

	batch := syncv1.SyncBatch{
		Version:  syncv1.Version,
		ClientID: w.clientID,
		Ticks:    ticks,
	}

	ack, err := w.backend.Sync(ctx, batch)
	if err != nil {
		return err
	}

	log.Printf("syncworker: sent %d ticks (accepted=%d duplicates=%d)", len(ticks), ack.Accepted, ack.Duplicates)
	return w.store.MarkSent(ctx, ids)
}
