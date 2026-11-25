package notify

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/backend/domain"
)

// fakeSender записывает отправленные сообщения.
type fakeSender struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakeSender) Name() string { return "fake" }

func (f *fakeSender) Send(_ context.Context, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, text)
	return nil
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func raised(typ domain.AlertType) domain.Alert {
	return domain.Alert{Type: typ, Severity: domain.SeverityWarning, Message: "test", At: time.Now(), State: domain.AlertRaised}
}

func cleared(typ domain.AlertType) domain.Alert {
	return domain.Alert{Type: typ, Severity: domain.SeverityWarning, Message: "test", At: time.Now(), State: domain.AlertCleared}
}

func TestNotify_RaisedOnceWithinCooldown(t *testing.T) {
	s := &fakeSender{}
	u := New(s, time.Hour, true)
	ctx := context.Background()

	// Три raised подряд в пределах cooldown → только одно сообщение.
	for i := 0; i < 3; i++ {
		if err := u.Notify(ctx, "edge-01", []domain.Alert{raised(domain.AlertO2Low)}); err != nil {
			t.Fatalf("notify: %v", err)
		}
	}
	if got := s.count(); got != 1 {
		t.Fatalf("want 1 message, got %d", got)
	}
}

func TestNotify_DifferentTypesIndependent(t *testing.T) {
	s := &fakeSender{}
	u := New(s, time.Hour, true)
	ctx := context.Background()

	u.Notify(ctx, "edge-01", []domain.Alert{raised(domain.AlertO2Low)})
	u.Notify(ctx, "edge-01", []domain.Alert{raised(domain.AlertPHHigh)})

	if got := s.count(); got != 2 {
		t.Fatalf("want 2 messages (different types), got %d", got)
	}
}

func TestNotify_ClearedThenRaisedAgain(t *testing.T) {
	s := &fakeSender{}
	u := New(s, time.Hour, true)
	ctx := context.Background()

	u.Notify(ctx, "edge-01", []domain.Alert{raised(domain.AlertO2Low)})   // 1: raised
	u.Notify(ctx, "edge-01", []domain.Alert{cleared(domain.AlertO2Low)})  // 2: cleared
	u.Notify(ctx, "edge-01", []domain.Alert{raised(domain.AlertO2Low)})   // 3: raised снова (cooldown сброшен)

	if got := s.count(); got != 3 {
		t.Fatalf("want 3 messages, got %d", got)
	}
}

func TestNotify_ClearedWithoutRaised_NoMessage(t *testing.T) {
	s := &fakeSender{}
	u := New(s, time.Hour, true)
	ctx := context.Background()

	// cleared без предшествующего raised → не отправляем.
	u.Notify(ctx, "edge-01", []domain.Alert{cleared(domain.AlertO2Low)})
	if got := s.count(); got != 0 {
		t.Fatalf("want 0 messages, got %d", got)
	}
}

func TestNotify_ClearedDisabled(t *testing.T) {
	s := &fakeSender{}
	u := New(s, time.Hour, false) // notifyCleared = false
	ctx := context.Background()

	u.Notify(ctx, "edge-01", []domain.Alert{raised(domain.AlertO2Low)})
	u.Notify(ctx, "edge-01", []domain.Alert{cleared(domain.AlertO2Low)})

	if got := s.count(); got != 1 {
		t.Fatalf("want 1 message (cleared disabled), got %d", got)
	}
}
