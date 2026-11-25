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
		want    bool
	}{
		{
			name: "нет данных — Safe Mode",
			ok:   false,
			want: true,
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
