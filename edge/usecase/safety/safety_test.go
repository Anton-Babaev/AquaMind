package safety

import (
	"context"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// fakeClock — поддельный ReadingClock.
type fakeClock struct {
	reading domain.SensorReading
	ok      bool
}

func (f *fakeClock) LastReading() (domain.SensorReading, bool) { return f.reading, f.ok }

// fakeController — поддельный SafeModeController, запоминает последнее значение.
type fakeController struct {
	last bool
	set  bool
}

func (f *fakeController) SetSafeMode(_ context.Context, on bool) error {
	f.last = on
	f.set = true
	return nil
}

func TestCheck(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	staleAfter := 10 * time.Second

	tests := []struct {
		name    string
		reading domain.SensorReading
		ok      bool
		elapsed time.Duration // сколько прошло с момента запуска (первой проверки)
		want    bool
	}{
		{
			name:    "нет данных, только стартовали — grace period, не Safe Mode",
			ok:      false,
			elapsed: 0,
			want:    false,
		},
		{
			name:    "нет данных, grace period истёк — Safe Mode",
			ok:      false,
			elapsed: staleAfter,
			want:    true,
		},
		{
			name:    "свежие данные — норма",
			reading: domain.SensorReading{At: now.Add(-2 * time.Second)},
			ok:      true,
			want:    false,
		},
		{
			name:    "ровно на границе — норма (не строго больше)",
			reading: domain.SensorReading{At: now.Add(-staleAfter)},
			ok:      true,
			want:    false,
		},
		{
			name:    "устаревшие данные — Safe Mode",
			reading: domain.SensorReading{At: now.Add(-11 * time.Second)},
			ok:      true,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := &fakeClock{reading: tt.reading, ok: tt.ok}
			ctrl := &fakeController{}
			u := New(clock, ctrl, staleAfter)
			// startedAt выставляем в прошлое на elapsed, чтобы смоделировать
			// прошедшее с запуска время.
			u.startedAt = now.Add(-tt.elapsed)
			u.now = func() time.Time { return now }

			got, err := u.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
			if !ctrl.set {
				t.Errorf("SetSafeMode не был вызван")
			}
			if ctrl.last != tt.want {
				t.Errorf("SetSafeMode(%v), want %v", ctrl.last, tt.want)
			}
		})
	}
}
