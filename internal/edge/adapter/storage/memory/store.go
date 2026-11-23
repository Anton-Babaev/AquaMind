package memory

import (
	"context"
	"sync"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// Store — временная in-memory реализация порта Store для Этапа 2.
// На Этапе 3 заменяется на SQLiteRepository. Реализует узкие интерфейсы
// control.Store и settings.Store.
type Store struct {
	mu         sync.RWMutex
	thresholds domain.Thresholds
	readings   []domain.SensorReading
	alerts     []domain.Alert
}

// New создаёт in-memory хранилище с заданными порогами.
func New(thresholds domain.Thresholds) *Store {
	return &Store{thresholds: thresholds}
}

// SaveTick сохраняет тик (в памяти — просто складывает в срезы).
func (s *Store) SaveTick(ctx context.Context, r domain.SensorReading, st domain.ActuatorState, alerts []domain.Alert) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readings = append(s.readings, r)
	s.alerts = append(s.alerts, alerts...)
	return nil
}

// LoadThresholds возвращает текущие пороги.
func (s *Store) LoadThresholds(ctx context.Context) (domain.Thresholds, error) {
	if err := ctx.Err(); err != nil {
		return domain.Thresholds{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.thresholds, nil
}

// SaveThresholds обновляет пороги (для settings).
func (s *Store) SaveThresholds(ctx context.Context, t domain.Thresholds) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.thresholds = t
	return nil
}

// Readings возвращает сохранённые показания (для тестов/отладки).
func (s *Store) Readings() []domain.SensorReading {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.SensorReading(nil), s.readings...)
}

// Alerts возвращает сохранённые алерты (для тестов/отладки).
func (s *Store) Alerts() []domain.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.Alert(nil), s.alerts...)
}
