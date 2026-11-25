package safety

import (
	"context"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// ReadingClock — источник времени последнего успешного чтения датчиков.
// Реализуется ControlUsecase (метод LastReading).
type ReadingClock interface {
	// LastReading возвращает последнее показание и признак того,
	// было ли уже хотя бы одно чтение.
	LastReading() (domain.SensorReading, bool)
}

// SafeModeController — переключатель безопасного режима.
// Реализуется ControlUsecase (метод SetSafeMode).
type SafeModeController interface {
	SetSafeMode(ctx context.Context, on bool) error
}
