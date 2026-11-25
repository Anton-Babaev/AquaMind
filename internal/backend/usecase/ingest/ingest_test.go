package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/backend/domain"
	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
)

// fakeStore — хранилище-заглушка: запоминает вызовы.
type fakeStore struct {
	accepted   int
	duplicates int
	gotClient  string
	gotTicks   []domain.Tick
}

func (f *fakeStore) SaveBatch(_ context.Context, clientID string, ticks []domain.Tick) (int, int, error) {
	f.gotClient = clientID
	f.gotTicks = ticks
	return f.accepted, f.duplicates, nil
}

// fakeNotifier — уведомления-заглушка.
type fakeNotifier struct {
	called    bool
	gotAlerts []domain.Alert
}

func (f *fakeNotifier) Notify(_ context.Context, _ string, alerts []domain.Alert) error {
	f.called = true
	f.gotAlerts = alerts
	return nil
}

func TestHandle_ConvertsAndSaves(t *testing.T) {
	store := &fakeStore{accepted: 2}
	notifier := &fakeNotifier{}
	u := New(store, notifier)

	now := time.Now().UTC()
	batch := syncv1.SyncBatch{
		Version:  1,
		ClientID: "edge-01",
		Ticks: []syncv1.TickDTO{
			{
				Reading:  syncv1.ReadingDTO{TempC: 24.5, FlowLPM: 10, O2MgL: 7, PH: 7.2, At: now},
				Actuator: syncv1.ActuatorDTO{Heating: true, Normal: true},
				Alerts: []syncv1.AlertDTO{
					{Type: "o2_low", Severity: "warning", Message: "low o2", At: now, State: "raised"},
				},
			},
			{
				Reading: syncv1.ReadingDTO{TempC: 25, FlowLPM: 10, O2MgL: 7, PH: 7.2, At: now.Add(time.Second)},
			},
		},
	}

	ack, err := u.Handle(context.Background(), batch)
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if ack.Accepted != 2 {
		t.Fatalf("want accepted=2, got %d", ack.Accepted)
	}
	if store.gotClient != "edge-01" {
		t.Fatalf("want client edge-01, got %s", store.gotClient)
	}
	if len(store.gotTicks) != 2 {
		t.Fatalf("want 2 ticks, got %d", len(store.gotTicks))
	}
	if store.gotTicks[0].Reading.TempC != 24.5 {
		t.Fatalf("want temp 24.5, got %v", store.gotTicks[0].Reading.TempC)
	}
	if !store.gotTicks[0].Actuator.Heating {
		t.Fatal("want heating=true")
	}
	if len(store.gotTicks[0].Alerts) != 1 {
		t.Fatalf("want 1 alert, got %d", len(store.gotTicks[0].Alerts))
	}
	if !notifier.called {
		t.Fatal("want notifier called")
	}
	if len(notifier.gotAlerts) != 1 {
		t.Fatalf("want 1 alert to notifier, got %d", len(notifier.gotAlerts))
	}
}

func TestHandle_NoAlerts_NotifierNotCalled(t *testing.T) {
	store := &fakeStore{accepted: 1}
	notifier := &fakeNotifier{}
	u := New(store, notifier)

	batch := syncv1.SyncBatch{
		ClientID: "edge-01",
		Ticks: []syncv1.TickDTO{
			{Reading: syncv1.ReadingDTO{At: time.Now()}},
		},
	}
	if _, err := u.Handle(context.Background(), batch); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if notifier.called {
		t.Fatal("notifier should not be called when no alerts")
	}
}

func TestHandle_EmptyClientID_Error(t *testing.T) {
	u := New(&fakeStore{}, &fakeNotifier{})
	_, err := u.Handle(context.Background(), syncv1.SyncBatch{})
	if err == nil {
		t.Fatal("want error for empty client_id")
	}
}

func TestHandle_AllDuplicates_NotifierNotCalled(t *testing.T) {
	store := &fakeStore{accepted: 0, duplicates: 2}
	notifier := &fakeNotifier{}
	u := New(store, notifier)

	batch := syncv1.SyncBatch{
		ClientID: "edge-01",
		Ticks: []syncv1.TickDTO{
			{Reading: syncv1.ReadingDTO{At: time.Now()}, Alerts: []syncv1.AlertDTO{
				{Type: "o2_low", State: "raised", At: time.Now()},
			}},
		},
	}
	if _, err := u.Handle(context.Background(), batch); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if notifier.called {
		t.Fatal("notifier should not be called when all ticks are duplicates")
	}
}
