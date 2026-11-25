// Package watchdog — delivery-триггер сторожевого таймера.
//
// По таймеру вызывает safety.Usecase.Check, который решает, переводить ли
// систему в Safe Mode. Здесь только расписание и логирование переходов.
package watchdog

import (
	"context"
	"log"
	"time"
)

// Checker — то, что умеет выполнить одну проверку (safety.Usecase).
type Checker interface {
	Check(ctx context.Context) (bool, error)
}

// Watchdog — delivery-компонент: периодически проверяет свежесть данных.
type Watchdog struct {
	checker  Checker
	interval time.Duration
}

// New создаёт Watchdog.
func New(checker Checker, interval time.Duration) *Watchdog {
	return &Watchdog{checker: checker, interval: interval}
}

// Run запускает цикл проверок до отмены контекста.
func (w *Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	prevSafe := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			safe, err := w.checker.Check(ctx)
			if err != nil {
				log.Printf("watchdog: check error: %v", err)
				continue
			}
			// Логируем только переходы, чтобы не засорять журнал.
			if safe != prevSafe {
				if safe {
					log.Printf("watchdog: данные устарели — Safe Mode ВКЛ")
				} else {
					log.Printf("watchdog: данные свежие — Safe Mode ВЫКЛ")
				}
				prevSafe = safe
			}
		}
	}
}
