//go:build linux

package gpio

import (
	"fmt"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

// New открывает GPIO и создаёт адаптер на реальных пинах Raspberry Pi.
// Требует прав root (или членства в группе gpio) — go-rpio маппит /dev/gpiomem.
// pins — раскладка BCM-пинов; для схемы по умолчанию используйте DefaultPins.
func New(pins Pins) (*LEDs, error) {
	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("open gpio (need root / group gpio?): %w", err)
	}
	return newLEDs(&rpioBackend{}, pins)
}

// rpioBackend — backend поверх go-rpio.
type rpioBackend struct{}

// Configure переводит пины в режим выхода и гасит их.
func (b *rpioBackend) Configure(pins Pins) error {
	for _, p := range []int{pins.Heating, pins.Alarm, pins.Normal} {
		pin := rpio.Pin(p)
		pin.Output()
		pin.Low()
	}
	return nil
}

// Write выставляет уровень на пине.
func (b *rpioBackend) Write(pin int, on bool) error {
	p := rpio.Pin(pin)
	if on {
		p.High()
	} else {
		p.Low()
	}
	return nil
}

// Close освобождает GPIO (unmap регистров).
func (b *rpioBackend) Close() error {
	return rpio.Close()
}
