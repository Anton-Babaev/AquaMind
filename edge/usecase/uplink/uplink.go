// Package uplink реализует сценарий отправки накопленных данных на Backend.
//
// Usecase отвечает за логику: забрать пачку из outbox, собрать SyncBatch,
// отправить, пометить отправленные. Delivery (syncworker) лишь вызывает
// SyncOnce по таймеру и логирует результат — он не знает деталей протокола.
//
// Гарантия «at-least-once»: записи помечаются отправленными ТОЛЬКО после
// успешного ответа Backend. Если отправка упала — записи остаются в outbox
// и будут отправлены повторно. Дубликаты на стороне Backend отсекаются по
// (client_id, at) — см. backend/usecase/ingest.
package uplink

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Anton-Babaev/AquaMind/contract/syncv1"
)

// Usecase — сценарий синхронизации.
type Usecase struct {
	outbox   Outbox
	backend  Backend
	clientID string
	batch    int
}

// New создаёт Usecase.
// batch — максимум записей в одном пакете (<=0 → 100).
func New(outbox Outbox, backend Backend, clientID string, batch int) *Usecase {
	if batch <= 0 {
		batch = 100
	}
	return &Usecase{outbox: outbox, backend: backend, clientID: clientID, batch: batch}
}

// SyncOnce выполняет одну попытку отправки.
// Возвращает количество отправленных тиков (0, если отправлять нечего).
func (u *Usecase) SyncOnce(ctx context.Context) (int, error) {
	entries, err := u.outbox.PendingOutbox(ctx, u.batch)
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}

	ticks := make([]syncv1.TickDTO, 0, len(entries))
	ids := make([]int64, 0, len(entries))
	for _, e := range entries {
		var t syncv1.TickDTO
		if err := json.Unmarshal([]byte(e.Payload), &t); err != nil {
			// Битую запись помечаем отправленной, чтобы не зацикливаться на ней.
			log.Printf("uplink: skip bad payload id=%d: %v", e.ID, err)
			ids = append(ids, e.ID)
			continue
		}
		ticks = append(ticks, t)
		ids = append(ids, e.ID)
	}

	// Если все записи оказались битыми — просто помечаем их и выходим.
	if len(ticks) == 0 {
		return 0, u.outbox.MarkSent(ctx, ids)
	}

	batch := syncv1.SyncBatch{
		Version:  syncv1.Version,
		ClientID: u.clientID,
		Ticks:    ticks,
	}

	ack, err := u.backend.Sync(ctx, batch)
	if err != nil {
		return 0, fmt.Errorf("send batch: %w", err)
	}

	// Помечаем отправленными только после успешного ответа.
	if err := u.outbox.MarkSent(ctx, ids); err != nil {
		return 0, fmt.Errorf("mark sent: %w", err)
	}

	log.Printf("uplink: sent %d ticks (accepted=%d duplicates=%d)", len(ticks), ack.Accepted, ack.Duplicates)
	return len(ticks), nil
}
