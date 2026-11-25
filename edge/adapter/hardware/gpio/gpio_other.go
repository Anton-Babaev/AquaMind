//go:build !linux

package gpio

import (
	"fmt"
	"runtime"
)

// New на не-Linux платформах (ноутбук) возвращает ошибку.
// Это позволяет собирать и тестировать проект на Windows/macOS без железа:
// реальный GPIO доступен только на Raspberry Pi (linux/arm).
func New(pins Pins) (*LEDs, error) {
	return nil, fmt.Errorf("gpio adapter is only available on linux/arm (Raspberry Pi), current platform: %s/%s", runtime.GOOS, runtime.GOARCH)
}
