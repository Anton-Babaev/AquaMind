package alert

import (
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

func mkAlert(typ domain.AlertType) domain.Alert {
	return domain.Alert{
		Type:     typ,
		Severity: domain.SeverityCritical,
		Message:  string(typ),
		At:       time.Now().UTC(),
	}
}

func countState(events []domain.Alert, state domain.AlertState) int {
	n := 0
	for _, e := range events {
		if e.State == state {
			n++
		}
	}
	return n
}

func TestTracker_RaiseOnce(t *testing.T) {
	tr := NewTracker()

	// Первый тик: алерт появился → raised.
	ev := tr.Track([]domain.Alert{mkAlert(domain.AlertO2Low)})
	if len(ev) != 1 || ev[0].State != domain.AlertRaised {
		t.Fatalf("expected 1 raised, got %+v", ev)
	}

	// Второй тик: тот же алерт → нет событий (антиспам).
	ev = tr.Track([]domain.Alert{mkAlert(domain.AlertO2Low)})
	if len(ev) != 0 {
		t.Fatalf("expected no events on repeat, got %+v", ev)
	}
}

func TestTracker_Clear(t *testing.T) {
	tr := NewTracker()
	tr.Track([]domain.Alert{mkAlert(domain.AlertO2Low)})

	// Алерт пропал → cleared.
	ev := tr.Track(nil)
	if len(ev) != 1 || ev[0].State != domain.AlertCleared {
		t.Fatalf("expected 1 cleared, got %+v", ev)
	}

	// Повторно пусто → нет событий.
	ev = tr.Track(nil)
	if len(ev) != 0 {
		t.Fatalf("expected no events, got %+v", ev)
	}
}

func TestTracker_MultipleAlerts(t *testing.T) {
	tr := NewTracker()

	// Два алерта появились.
	ev := tr.Track([]domain.Alert{mkAlert(domain.AlertO2Low), mkAlert(domain.AlertNoFlow)})
	if countState(ev, domain.AlertRaised) != 2 {
		t.Fatalf("expected 2 raised, got %+v", ev)
	}

	// Один пропал, один остался → 1 cleared.
	ev = tr.Track([]domain.Alert{mkAlert(domain.AlertO2Low)})
	if countState(ev, domain.AlertCleared) != 1 {
		t.Fatalf("expected 1 cleared, got %+v", ev)
	}
	if countState(ev, domain.AlertRaised) != 0 {
		t.Fatalf("expected 0 raised, got %+v", ev)
	}
}

func TestTracker_Active(t *testing.T) {
	tr := NewTracker()
	tr.Track([]domain.Alert{mkAlert(domain.AlertO2Low), mkAlert(domain.AlertNoFlow)})
	if got := len(tr.Active()); got != 2 {
		t.Fatalf("Active() = %d, want 2", got)
	}
}
