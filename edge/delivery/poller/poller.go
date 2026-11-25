package poller

import (
	"context"
	"log"
	"time"
)

// TickRunner — то, что умеет выполнять один цикл (ControlUsecase).
type TickRunner interface {
	Tick(ctx context.Context) error
}

// Poller — delivery-триггер: по тику вызывает TickRunner.
type Poller struct {
	runner   TickRunner
	interval time.Duration
}

// New создаёт Poller.
func New(runner TickRunner, interval time.Duration) *Poller {
	return &Poller{runner: runner, interval: interval}
}

// Run запускает цикл опроса до отмены контекста.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.runner.Tick(ctx); err != nil {
				log.Printf("poller: tick error: %v", err)
			}
		}
	}
}
