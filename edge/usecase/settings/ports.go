package settings

import (
	"context"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// Store — порт хранения порогов.
// Реализуется SQLite-репозиторием.
type Store interface {
	LoadThresholds(ctx context.Context) (domain.Thresholds, error)
	SaveThresholds(ctx context.Context, t domain.Thresholds) error
}
