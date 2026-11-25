// Package safety реализует сторожевой таймер (watchdog) Edge-контроллера.
//
// Идея: если данные от датчиков «устарели» (последнее успешное чтение старше
// StaleAfter) или чтений ещё не было вовсе, система переходит в Safe Mode —
// подогрев выключается, поднимается критический алерт. Как только данные снова
// становятся свежими, Safe Mode снимается.
//
// Usecase не знает, КАК читаются датчики и КАК выставляются выходы: он работает
// через узкие порты ReadingClock и SafeModeController (их реализует ControlUsecase).
// Триггер по таймеру (как часто проверять) — в delivery/watchdog.
package safety

import (
	"context"
	"time"
)

// Usecase — сторожевой таймер.
type Usecase struct {
	clock      ReadingClock
	controller SafeModeController
	staleAfter time.Duration
	now        func() time.Time // подменяемое время (для тестов)

	startedAt time.Time // момент первой проверки (для grace period на старте)
}

// New создаёт Usecase.
// staleAfter — через сколько после последнего чтения данные считаются устаревшими.
func New(clock ReadingClock, controller SafeModeController, staleAfter time.Duration) *Usecase {
	return &Usecase{
		clock:      clock,
		controller: controller,
		staleAfter: staleAfter,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

// Check выполняет одну проверку «свежести» данных и при необходимости
// переключает Safe Mode. Возвращает true, если система сейчас в Safe Mode.
//
// Правила:
//   - чтений ещё не было, но с запуска прошло < staleAfter → grace period,
//     Safe Mode НЕ включаем (система только стартовала, первый Tick ещё не успел);
//   - чтений ещё не было, и прошло >= staleAfter → Safe Mode (данных нет);
//   - последнее чтение старше порога → Safe Mode (данные устарели);
//   - иначе                          → нормальный режим.
func (u *Usecase) Check(ctx context.Context) (bool, error) {
	now := u.now()
	if u.startedAt.IsZero() {
		u.startedAt = now
	}

	reading, ok := u.clock.LastReading()

	safe := false
	if !ok {
		// Данных ещё не было. Даём grace period на старте: первый Tick поллера
		// может занять время (инициализация SQLite на SD-карте и т.п.), и без
		// этого watchdog ложно включает Safe Mode сразу после запуска.
		safe = now.Sub(u.startedAt) >= u.staleAfter
	} else if now.Sub(reading.At) > u.staleAfter {
		// Данные устарели.
		safe = true
	}

	if err := u.controller.SetSafeMode(ctx, safe); err != nil {
		return safe, err
	}
	return safe, nil
}
