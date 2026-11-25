package console

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// LEDs — реализация порта Actuators для разработки на ноутбуке.
// Рисует состояние светодиодов в терминале вместо GPIO.
type LEDs struct {
	mu sync.Mutex
	w  io.Writer
}

// New создаёт console-адаптер, пишущий в stdout.
func New() *LEDs {
	return &LEDs{w: os.Stdout}
}

// NewWithWriter создаёт адаптер с произвольным writer (для тестов).
func NewWithWriter(w io.Writer) *LEDs {
	return &LEDs{w: w}
}

// Apply печатает состояние светодиодов.
func (l *LEDs) Apply(ctx context.Context, s domain.ActuatorState) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	yellow := "⚪"
	if s.Heating {
		yellow = "🟡"
	}
	red := "⚪"
	if s.Alarm {
		red = "🔴"
	}
	green := "⚪"
	if s.Normal {
		green = "🟢"
	}

	safe := ""
	if s.SafeMode {
		safe = "  [SAFE MODE]"
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := fmt.Fprintf(l.w, "LED  подогрев:%s  тревога:%s  норма:%s%s\n", yellow, red, green, safe)
	return err
}
