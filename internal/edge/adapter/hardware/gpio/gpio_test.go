package gpio

import (
	"context"
	"errors"
	"testing"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// fakeBackend — фейковый backend для тестов без железа.
type fakeBackend struct {
	configured bool
	closed     bool
	levels     map[int]bool
	writeErr   error
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{levels: make(map[int]bool)}
}

func (f *fakeBackend) Configure(pins Pins) error {
	f.configured = true
	// Как и реальный backend — гасим все пины при конфигурации.
	f.levels[pins.Heating] = false
	f.levels[pins.Alarm] = false
	f.levels[pins.Normal] = false
	return nil
}

func (f *fakeBackend) Write(pin int, on bool) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.levels[pin] = on
	return nil
}

func (f *fakeBackend) Close() error {
	f.closed = true
	return nil
}

func TestPins_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pins    Pins
		wantErr bool
	}{
		{"default ok", DefaultPins, false},
		{"negative", Pins{Heating: -1, Alarm: 27, Normal: 22}, true},
		{"duplicate", Pins{Heating: 17, Alarm: 17, Normal: 22}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pins.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewLEDs_ConfiguresPins(t *testing.T) {
	fb := newFakeBackend()
	l, err := newLEDs(fb, DefaultPins)
	if err != nil {
		t.Fatalf("newLEDs: %v", err)
	}
	if !fb.configured {
		t.Error("backend must be configured on creation")
	}
	if l == nil {
		t.Fatal("nil adapter")
	}
}

func TestNewLEDs_InvalidPins(t *testing.T) {
	fb := newFakeBackend()
	if _, err := newLEDs(fb, Pins{Heating: 17, Alarm: 17, Normal: 22}); err == nil {
		t.Fatal("expected error for duplicate pins")
	}
}

func TestApply_SetsLevels(t *testing.T) {
	fb := newFakeBackend()
	l, err := newLEDs(fb, DefaultPins)
	if err != nil {
		t.Fatalf("newLEDs: %v", err)
	}

	// Подогрев включён, тревоги нет, норма.
	state := domain.ActuatorState{Heating: true, Alarm: false, Normal: true}
	if err := l.Apply(context.Background(), state); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !fb.levels[DefaultPins.Heating] {
		t.Error("heating pin should be high")
	}
	if fb.levels[DefaultPins.Alarm] {
		t.Error("alarm pin should be low")
	}
	if !fb.levels[DefaultPins.Normal] {
		t.Error("normal pin should be high")
	}

	// Тревога: подогрев выкл, красный вкл, зелёный выкл.
	state = domain.ActuatorState{Heating: false, Alarm: true, Normal: false}
	if err := l.Apply(context.Background(), state); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if fb.levels[DefaultPins.Heating] {
		t.Error("heating pin should be low")
	}
	if !fb.levels[DefaultPins.Alarm] {
		t.Error("alarm pin should be high")
	}
	if fb.levels[DefaultPins.Normal] {
		t.Error("normal pin should be low")
	}
}

func TestApply_ContextCancelled(t *testing.T) {
	fb := newFakeBackend()
	l, err := newLEDs(fb, DefaultPins)
	if err != nil {
		t.Fatalf("newLEDs: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.Apply(ctx, domain.ActuatorState{}); err == nil {
		t.Fatal("expected context error")
	}
}

func TestApply_WriteError(t *testing.T) {
	fb := newFakeBackend()
	fb.writeErr = errors.New("boom")
	l, err := newLEDs(fb, DefaultPins)
	if err != nil {
		t.Fatalf("newLEDs: %v", err)
	}
	if err := l.Apply(context.Background(), domain.ActuatorState{Heating: true}); err == nil {
		t.Fatal("expected write error")
	}
}

func TestClose_TurnsOffAndCloses(t *testing.T) {
	fb := newFakeBackend()
	l, err := newLEDs(fb, DefaultPins)
	if err != nil {
		t.Fatalf("newLEDs: %v", err)
	}

	// Включаем всё, затем закрываем — всё должно погаснуть.
	_ = l.Apply(context.Background(), domain.ActuatorState{Heating: true, Alarm: true, Normal: true})
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !fb.closed {
		t.Error("backend must be closed")
	}
	if fb.levels[DefaultPins.Heating] || fb.levels[DefaultPins.Alarm] || fb.levels[DefaultPins.Normal] {
		t.Error("all pins must be low after Close")
	}

	// Повторный Close — без ошибки (идемпотентность).
	if err := l.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}

	// Apply после Close — ошибка.
	if err := l.Apply(context.Background(), domain.ActuatorState{}); err == nil {
		t.Error("Apply after Close must fail")
	}
}
