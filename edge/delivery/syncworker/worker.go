// Package syncworker — delivery-триггер синхронизации.
//
// По таймеру вызывает uplink.Usecase.SyncOnce. Вся логика (что забирать,
// как батчить, когда помечать отправленным) — в usecase; здесь только
// расписание и логирование ошибок.
package syncworker

import (
	"context"
	"log"
	"time"
)

// Syncer — то, что умеет выполнить одну попытку отправки (uplink.Usecase).
type Syncer interface {
	SyncOnce(ctx context.Context) (int, error)
}

// Worker — delivery-компонент: периодически запускает синхронизацию.
type Worker struct {
	syncer   Syncer
	interval time.Duration
}

// New создаёт SyncWorker.
func New(syncer Syncer, interval time.Duration) *Worker {
	return &Worker{syncer: syncer, interval: interval}
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
			if _, err := w.syncer.SyncOnce(ctx); err != nil {
				log.Printf("синхронизация: %v", err)
			}
		}
	}
}
