// Package retention — delivery-триггер очистки устаревших данных.
//
// По таймеру вызывает Purger.Purge, который удаляет readings/alerts и
// отправленные outbox старше retention. Логика «что и как чистить» — в
// адаптере хранилища; здесь только расписание и логирование.
package retention

import (
	"context"
	"log"
	"time"
)

// Purger — то, что умеет выполнить очистку (sqlite.Repository.Purge).
type Purger interface {
	Purge(ctx context.Context, retention time.Duration) (readings, alerts, outbox int64, err error)
}

// Worker — delivery-компонент периодической очистки.
type Worker struct {
	purger    Purger
	retention time.Duration // сколько данных хранить
	interval  time.Duration // как часто запускать очистку
}

// New создаёт Worker.
// retention — срок хранения данных; interval — период запуска очистки.
func New(purger Purger, retention, interval time.Duration) *Worker {
	return &Worker{purger: purger, retention: retention, interval: interval}
}

// Run запускает цикл очистки до отмены контекста.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			readings, alerts, outbox, err := w.purger.Purge(ctx, w.retention)
			if err != nil {
				log.Printf("retention: purge error: %v", err)
				continue
			}
			if readings+alerts+outbox > 0 {
				log.Printf("retention: удалено readings=%d alerts=%d outbox=%d", readings, alerts, outbox)
			}
		}
	}
}
