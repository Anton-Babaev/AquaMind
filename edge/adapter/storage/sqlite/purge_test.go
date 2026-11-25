package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

func TestPurge(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Старый тик (2 часа назад) и свежий (сейчас).
	old := domain.SensorReading{TempC: 24, FlowLPM: 10, O2MgL: 7, PH: 7, At: now.Add(-2 * time.Hour)}
	fresh := domain.SensorReading{TempC: 25, FlowLPM: 11, O2MgL: 7, PH: 7, At: now}

	if err := repo.SaveTick(ctx, old, domain.ActuatorState{Normal: true}, nil); err != nil {
		t.Fatalf("SaveTick(old): %v", err)
	}
	if err := repo.SaveTick(ctx, fresh, domain.ActuatorState{Normal: true}, nil); err != nil {
		t.Fatalf("SaveTick(fresh): %v", err)
	}

	// Помечаем ВСЕ outbox отправленными, чтобы проверить очистку отправленных.
	entries, err := repo.PendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("PendingOutbox: %v", err)
	}
	ids := make([]int64, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	if err := repo.MarkSent(ctx, ids); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}

	// Retention = 1 час: старый тик должен удалиться, свежий — остаться.
	readings, alerts, outbox, err := repo.Purge(ctx, time.Hour)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if readings != 1 {
		t.Errorf("удалено readings = %d, want 1", readings)
	}
	_ = alerts
	// Обе outbox-записи отправлены только что (sent_at_ms = сейчас),
	// поэтому по retention=1h они НЕ удаляются.
	if outbox != 0 {
		t.Errorf("удалено outbox = %d, want 0 (отправлены только что)", outbox)
	}

	// Проверяем, что свежее чтение осталось: pending outbox пуст (всё отправлено),
	// поэтому проверяем через повторный SaveTick+счётчик не будем — достаточно
	// убедиться, что Purge не удалил свежие данные косвенно через readings=1.
}

func TestPurge_ZeroRetention_Noop(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	if err := repo.SaveTick(ctx, testReading(), domain.ActuatorState{Normal: true}, nil); err != nil {
		t.Fatalf("SaveTick: %v", err)
	}

	r, a, o, err := repo.Purge(ctx, 0)
	if err != nil {
		t.Fatalf("Purge(0): %v", err)
	}
	if r != 0 || a != 0 || o != 0 {
		t.Errorf("Purge(0) удалил %d/%d/%d, want 0/0/0", r, a, o)
	}
}

func TestPurge_KeepsUnsentOutbox(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	// Старый тик, outbox НЕ помечаем отправленным.
	old := domain.SensorReading{TempC: 24, FlowLPM: 10, O2MgL: 7, PH: 7, At: time.Now().UTC().Add(-2 * time.Hour)}
	if err := repo.SaveTick(ctx, old, domain.ActuatorState{Normal: true}, nil); err != nil {
		t.Fatalf("SaveTick: %v", err)
	}

	// Retention = 1 час. readings удалится, но неотправленный outbox — нет.
	readings, _, outbox, err := repo.Purge(ctx, time.Hour)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if readings != 1 {
		t.Errorf("удалено readings = %d, want 1", readings)
	}
	if outbox != 0 {
		t.Errorf("удалено outbox = %d, want 0 (неотправленные сохраняем)", outbox)
	}

	// Неотправленная запись должна остаться в outbox.
	pending, err := repo.PendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("PendingOutbox: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("pending outbox = %d, want 1 (неотправленное не удаляем)", len(pending))
	}
}
