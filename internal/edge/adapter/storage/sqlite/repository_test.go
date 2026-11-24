package sqlite

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// openTestDB открывает временную БД в t.TempDir() и закрывает её по завершении теста.
func openTestDB(t *testing.T) *Repository {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatalf("Open() returned unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	})
	return repo
}

func testReading() domain.SensorReading {
	return domain.SensorReading{
		TempC:   24.1,
		FlowLPM: 12.0,
		O2MgL:   7.2,
		PH:      7.2,
		At:      time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestOpen_AppliesMigrations(t *testing.T) {
	repo := openTestDB(t)

	// Если миграции применились, таблица settings доступна и пуста.
	th, err := repo.LoadThresholds(context.Background())
	if err != nil {
		t.Fatalf("LoadThresholds() on fresh DB: %v", err)
	}
	// На пустой БД возвращаются значения по умолчанию.
	if th.TempLow != 22.0 {
		t.Errorf("default TempLow = %v, want 22.0", th.TempLow)
	}
}

func TestSaveTick_WritesReadingAndOutbox(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	reading := testReading()
	state := domain.ActuatorState{Heating: true, Normal: true}
	alerts := []domain.Alert{
		{Type: domain.AlertO2Low, Severity: domain.SeverityWarning, Message: "низкий O2", At: reading.At, State: domain.AlertRaised},
	}

	if err := repo.SaveTick(ctx, reading, state, alerts); err != nil {
		t.Fatalf("SaveTick() returned unexpected error: %v", err)
	}

	// В outbox должна появиться одна неотправленная запись.
	entries, err := repo.PendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("PendingOutbox() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("PendingOutbox() len = %d, want 1", len(entries))
	}

	// Проверяем, что payload корректно сериализован.
	var tick syncv1.TickDTO
	if err := json.Unmarshal([]byte(entries[0].Payload), &tick); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	if tick.Reading.TempC != reading.TempC {
		t.Errorf("payload TempC = %v, want %v", tick.Reading.TempC, reading.TempC)
	}
	if !tick.Actuator.Heating {
		t.Error("payload Heating = false, want true")
	}
	if len(tick.Alerts) != 1 || tick.Alerts[0].Type != string(domain.AlertO2Low) {
		t.Errorf("payload alerts = %+v, want one o2_low alert", tick.Alerts)
	}
}

func TestMarkSent_RemovesFromPending(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.SaveTick(ctx, testReading(), domain.ActuatorState{Normal: true}, nil); err != nil {
			t.Fatalf("SaveTick() error: %v", err)
		}
	}

	entries, err := repo.PendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("PendingOutbox() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("PendingOutbox() len = %d, want 3", len(entries))
	}

	ids := []int64{entries[0].ID, entries[1].ID}
	if err := repo.MarkSent(ctx, ids); err != nil {
		t.Fatalf("MarkSent() error: %v", err)
	}

	remaining, err := repo.PendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("PendingOutbox() error: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("after MarkSent PendingOutbox() len = %d, want 1", len(remaining))
	}
	if remaining[0].ID != entries[2].ID {
		t.Errorf("remaining id = %d, want %d", remaining[0].ID, entries[2].ID)
	}
}

func TestMarkSent_EmptyIDs(t *testing.T) {
	repo := openTestDB(t)
	if err := repo.MarkSent(context.Background(), nil); err != nil {
		t.Errorf("MarkSent(nil) error = %v, want nil", err)
	}
}

func TestPendingOutbox_Limit(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := repo.SaveTick(ctx, testReading(), domain.ActuatorState{Normal: true}, nil); err != nil {
			t.Fatalf("SaveTick() error: %v", err)
		}
	}

	entries, err := repo.PendingOutbox(ctx, 2)
	if err != nil {
		t.Fatalf("PendingOutbox() error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("PendingOutbox(limit=2) len = %d, want 2", len(entries))
	}
}

func TestThresholds_RoundTrip(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	want := domain.Thresholds{
		TempLow:           21.5,
		TempLowHysteresis: 0.8,
		TempCriticalLow:   17.0,
		TempCriticalHigh:  29.0,
		O2Low:             4.5,
		O2LowHysteresis:   0.3,
		FlowMin:           6.0,
	}
	if err := repo.SaveThresholds(ctx, want); err != nil {
		t.Fatalf("SaveThresholds() error: %v", err)
	}

	got, err := repo.LoadThresholds(ctx)
	if err != nil {
		t.Fatalf("LoadThresholds() error: %v", err)
	}
	if got != want {
		t.Errorf("LoadThresholds() = %+v, want %+v", got, want)
	}
}

func TestSaveThresholds_Upsert(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	first := domain.Thresholds{TempLow: 22, TempLowHysteresis: 1, TempCriticalLow: 18, TempCriticalHigh: 30, O2Low: 5, O2LowHysteresis: 0.5, FlowMin: 5}
	if err := repo.SaveThresholds(ctx, first); err != nil {
		t.Fatalf("SaveThresholds() error: %v", err)
	}

	second := first
	second.TempLow = 23
	if err := repo.SaveThresholds(ctx, second); err != nil {
		t.Fatalf("SaveThresholds() upsert error: %v", err)
	}

	got, err := repo.LoadThresholds(ctx)
	if err != nil {
		t.Fatalf("LoadThresholds() error: %v", err)
	}
	if got.TempLow != 23 {
		t.Errorf("TempLow after upsert = %v, want 23", got.TempLow)
	}
}

func TestSaveTick_ContextCancelled(t *testing.T) {
	repo := openTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.SaveTick(ctx, testReading(), domain.ActuatorState{}, nil); err == nil {
		t.Fatal("SaveTick() error = nil, want context error")
	}
}
