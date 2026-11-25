package query

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/backend/domain"
)

// fakeReader — управляемый Reader для тестов.
type fakeReader struct {
	clients    []domain.Client
	clientsErr error

	tick    domain.Tick
	tickOK  bool
	tickErr error

	ticks    []domain.Tick
	ticksErr error
	// захватываем аргументы, чтобы проверить нормализацию limit.
	gotLimit int

	alerts    []domain.Alert
	alertsErr error
}

func (f *fakeReader) ListClients(ctx context.Context) ([]domain.Client, error) {
	return f.clients, f.clientsErr
}

func (f *fakeReader) LatestTick(ctx context.Context, clientID string) (domain.Tick, bool, error) {
	return f.tick, f.tickOK, f.tickErr
}

func (f *fakeReader) TicksRange(ctx context.Context, clientID string, from, to time.Time, limit int) ([]domain.Tick, error) {
	f.gotLimit = limit
	return f.ticks, f.ticksErr
}

func (f *fakeReader) Alerts(ctx context.Context, clientID string, limit int) ([]domain.Alert, error) {
	f.gotLimit = limit
	return f.alerts, f.alertsErr
}

func TestClients(t *testing.T) {
	r := &fakeReader{clients: []domain.Client{{ClientID: "edge-01"}}}
	u := New(r)

	got, err := u.Clients(context.Background())
	if err != nil {
		t.Fatalf("Clients() error: %v", err)
	}
	if len(got) != 1 || got[0].ClientID != "edge-01" {
		t.Errorf("Clients() = %+v, want one edge-01", got)
	}
}

func TestClients_Error(t *testing.T) {
	r := &fakeReader{clientsErr: errors.New("db down")}
	if _, err := New(r).Clients(context.Background()); err == nil {
		t.Fatal("Clients() error = nil, want error")
	}
}

func TestState_EmptyClientID(t *testing.T) {
	if _, _, err := New(&fakeReader{}).State(context.Background(), ""); err == nil {
		t.Fatal("State(\"\") error = nil, want error")
	}
}

func TestState_OK(t *testing.T) {
	r := &fakeReader{tick: domain.Tick{Reading: domain.Reading{TempC: 24}}, tickOK: true}
	tick, ok, err := New(r).State(context.Background(), "edge-01")
	if err != nil {
		t.Fatalf("State() error: %v", err)
	}
	if !ok || tick.Reading.TempC != 24 {
		t.Errorf("State() = %+v ok=%v, want temp 24 ok=true", tick, ok)
	}
}

func TestState_NotFound(t *testing.T) {
	r := &fakeReader{tickOK: false}
	_, ok, err := New(r).State(context.Background(), "edge-01")
	if err != nil {
		t.Fatalf("State() error: %v", err)
	}
	if ok {
		t.Error("State() ok = true, want false")
	}
}

func TestTicks_EmptyClientID(t *testing.T) {
	if _, err := New(&fakeReader{}).Ticks(context.Background(), "", time.Time{}, time.Time{}, 0); err == nil {
		t.Fatal("Ticks(\"\") error = nil, want error")
	}
}

func TestTicks_LimitNormalization(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantLimit int
	}{
		{"ноль → дефолт 500", 0, 500},
		{"отрицательный → дефолт 500", -5, 500},
		{"слишком большой → 500", 10000, 500},
		{"в пределах → как есть", 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeReader{}
			if _, err := New(r).Ticks(context.Background(), "edge-01", time.Time{}, time.Time{}, tt.limit); err != nil {
				t.Fatalf("Ticks() error: %v", err)
			}
			if r.gotLimit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", r.gotLimit, tt.wantLimit)
			}
		})
	}
}

func TestAlerts_EmptyClientID(t *testing.T) {
	if _, err := New(&fakeReader{}).Alerts(context.Background(), "", 0); err == nil {
		t.Fatal("Alerts(\"\") error = nil, want error")
	}
}

func TestAlerts_LimitNormalization(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantLimit int
	}{
		{"ноль → дефолт 100", 0, 100},
		{"слишком большой → 100", 5000, 100},
		{"в пределах → как есть", 50, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeReader{}
			if _, err := New(r).Alerts(context.Background(), "edge-01", tt.limit); err != nil {
				t.Fatalf("Alerts() error: %v", err)
			}
			if r.gotLimit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", r.gotLimit, tt.wantLimit)
			}
		})
	}
}
