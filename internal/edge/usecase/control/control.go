package control

import (
	"context"
	"sync"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
	"github.com/Anton-Babaev/AquaMind/internal/edge/domain/policy"
)

// ControlUsecase — единственный владелец выходов.
// Сценарий тика: прочитать датчики → применить политики → выставить выходы →
// сохранить одной транзакцией.
//
// Важно: блокировка НЕ держится во время чтения датчиков, иначе Safe Mode
// не включится при зависшем чтении (см. §5.2).
type ControlUsecase struct {
	sensors  SensorSource
	actuators Actuators
	store    Store
	tracker  AlertTracker

	mu       sync.RWMutex
	state    domain.ActuatorState // текущее состояние выходов
	safeMode bool                 // текущий режим
}

// New создаёт ControlUsecase.
func New(sensors SensorSource, actuators Actuators, store Store, tracker AlertTracker) *ControlUsecase {
	return &ControlUsecase{
		sensors:   sensors,
		actuators: actuators,
		store:     store,
		tracker:   tracker,
	}
}

// Tick выполняет один цикл управления.
func (c *ControlUsecase) Tick(ctx context.Context) error {
	// 1. Читаем датчики БЕЗ блокировки.
	reading, err := c.sensors.Read(ctx)
	if err != nil {
		return err
	}

	// 2. Снимок состояния под блокировкой (быстро).
	c.mu.RLock()
	prevHeating := c.state.Heating
	safeMode := c.safeMode
	c.mu.RUnlock()

	// 3. Пороги.
	thresholds, err := c.store.LoadThresholds(ctx)
	if err != nil {
		return err
	}

	// 4. Применяем политики (чистые функции).
	heating := policy.HeatingPolicy(reading, prevHeating, safeMode, thresholds)
	activeAlerts := policy.AlarmPolicy(reading, safeMode, thresholds)

	// 5. Жизненный цикл алертов: события переходов.
	events := c.tracker.Track(activeAlerts)

	// 6. Формируем состояние выходов.
	state := domain.ActuatorState{
		Heating:  heating,
		Alarm:    len(activeAlerts) > 0,
		Normal:   len(activeAlerts) == 0,
		SafeMode: safeMode,
	}

	// 7. Выставляем выходы.
	if err := c.actuators.Apply(ctx, state); err != nil {
		return err
	}

	// 8. Сохраняем одной транзакцией: показание + состояние + алерты + outbox.
	if err := c.store.SaveTick(ctx, reading, state, events); err != nil {
		return err
	}

	// 9. Обновляем внутреннее состояние.
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()

	return nil
}

// SetSafeMode переводит систему в безопасный режим (или выводит из него).
// Вызывается SafetyUsecase через узкий интерфейс SafeModeController.
func (c *ControlUsecase) SetSafeMode(ctx context.Context, on bool) error {
	c.mu.Lock()
	c.safeMode = on
	c.mu.Unlock()
	return nil
}

// State возвращает текущее состояние выходов (для индикации/тестов).
func (c *ControlUsecase) State() domain.ActuatorState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}
