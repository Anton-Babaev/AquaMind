// Package gpio — реализация порта control.Actuators на реальных выходах
// Raspberry Pi через библиотеку go-rpio (чистый Go, без CGO).
//
// Схема подключения (BCM-нумерация):
//
//	🟡 Жёлтый  — подогрев — GPIO17 (физ. пин 11)
//	🔴 Красный — тревога  — GPIO27 (физ. пин 13)
//	🟢 Зелёный — норма    — GPIO22 (физ. пин 15)
//	⚫ GND     — земля    — физ. пин 9
//
// Каждый светодиод подключается через токоограничивающий резистор
// 220–330 Ом: GPIO-пин → резистор → анод → катод → GND.
//
// Пакет разделён по платформам:
//   - gpio_linux.go — реальный backend на go-rpio (только linux/arm);
//   - gpio_other.go — заглушка для ноутбука (Windows/macOS), чтобы
//     `go build ./...` и `go test ./...` проходили без железа.
//
// Общая (тестируемая) логика — в этом файле: она не зависит от платформы
// и работает через узкий интерфейс pinBackend.
package gpio

import (
	"context"
	"fmt"
	"sync"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// DefaultPins — раскладка пинов по умолчанию (BCM), совпадает со схемой в docs.
var DefaultPins = Pins{
	Heating: 17, // 🟡 подогрев, физ. пин 11
	Alarm:   27, // 🔴 тревога,  физ. пин 13
	Normal:  22, // 🟢 норма,    физ. пин 15
}

// Pins — номера BCM-пинов для каждого светодиода.
type Pins struct {
	Heating int
	Alarm   int
	Normal  int
}

// Validate проверяет, что пины заданы и не совпадают.
func (p Pins) Validate() error {
	if p.Heating < 0 || p.Alarm < 0 || p.Normal < 0 {
		return fmt.Errorf("gpio pins must be non-negative: %+v", p)
	}
	if p.Heating == p.Alarm || p.Heating == p.Normal || p.Alarm == p.Normal {
		return fmt.Errorf("gpio pins must be distinct: %+v", p)
	}
	return nil
}

// pinBackend — минимальный интерфейс над go-rpio, чтобы логику Apply
// можно было тестировать без железа (fake в тестах).
type pinBackend interface {
	// Configure переводит пины в режим выхода и гасит их.
	Configure(pins Pins) error
	// Write выставляет уровень на пине (true = высокий = светодиод горит).
	Write(pin int, on bool) error
	// Close освобождает ресурсы (unmap регистров).
	Close() error
}

// LEDs — реализация порта control.Actuators поверх GPIO.
// Потокобезопасна: Apply может вызываться из разных горутин.
type LEDs struct {
	mu      sync.Mutex
	backend pinBackend
	pins    Pins
	closed  bool
}

// newLEDs собирает адаптер поверх заданного backend (для тестов и платформ).
func newLEDs(backend pinBackend, pins Pins) (*LEDs, error) {
	if err := pins.Validate(); err != nil {
		return nil, err
	}
	if err := backend.Configure(pins); err != nil {
		return nil, fmt.Errorf("configure gpio pins: %w", err)
	}
	return &LEDs{backend: backend, pins: pins}, nil
}

// Apply выставляет светодиоды согласно состоянию выходов.
//   - 🟡 подогрев — s.Heating
//   - 🔴 тревога  — s.Alarm
//   - 🟢 норма    — s.Normal
func (l *LEDs) Apply(ctx context.Context, s domain.ActuatorState) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return fmt.Errorf("gpio: adapter is closed")
	}

	if err := l.backend.Write(l.pins.Heating, s.Heating); err != nil {
		return fmt.Errorf("write heating pin %d: %w", l.pins.Heating, err)
	}
	if err := l.backend.Write(l.pins.Alarm, s.Alarm); err != nil {
		return fmt.Errorf("write alarm pin %d: %w", l.pins.Alarm, err)
	}
	if err := l.backend.Write(l.pins.Normal, s.Normal); err != nil {
		return fmt.Errorf("write normal pin %d: %w", l.pins.Normal, err)
	}
	return nil
}

// Close гасит все светодиоды и освобождает GPIO.
// Вызывается при graceful shutdown (SIGTERM) — железо уходит в безопасное
// состояние: все индикаторы выключены.
func (l *LEDs) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	// Гасим все пины перед освобождением (best effort).
	_ = l.backend.Write(l.pins.Heating, false)
	_ = l.backend.Write(l.pins.Alarm, false)
	_ = l.backend.Write(l.pins.Normal, false)

	return l.backend.Close()
}
