package uplink

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Anton-Babaev/AquaMind/contract/syncv1"
)

// fakeOutbox — поддельный outbox.
type fakeOutbox struct {
	entries []OutboxEntry
	marked  []int64
	markErr error
	pendErr error
}

func (f *fakeOutbox) PendingOutbox(context.Context, int) ([]OutboxEntry, error) {
	return f.entries, f.pendErr
}

func (f *fakeOutbox) MarkSent(_ context.Context, ids []int64) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.marked = append(f.marked, ids...)
	return nil
}

// fakeBackend — поддельный Backend.
type fakeBackend struct {
	ack    syncv1.SyncAck
	err    error
	called int
	got    syncv1.SyncBatch
}

func (f *fakeBackend) Sync(_ context.Context, b syncv1.SyncBatch) (syncv1.SyncAck, error) {
	f.called++
	f.got = b
	return f.ack, f.err
}

func tickPayload(t *testing.T, temp float64) string {
	t.Helper()
	raw, err := json.Marshal(syncv1.TickDTO{Reading: syncv1.ReadingDTO{TempC: temp}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(raw)
}

func TestSyncOnce_Empty(t *testing.T) {
	ob := &fakeOutbox{}
	be := &fakeBackend{}
	u := New(ob, be, "edge-01", 100)

	n, err := u.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}
	if n != 0 {
		t.Errorf("ожидалось 0 отправленных, получено %d", n)
	}
	if be.called != 0 {
		t.Errorf("Backend не должен вызываться при пустом outbox")
	}
}

func TestSyncOnce_Success(t *testing.T) {
	ob := &fakeOutbox{entries: []OutboxEntry{
		{ID: 1, Payload: tickPayload(t, 24)},
		{ID: 2, Payload: tickPayload(t, 25)},
	}}
	be := &fakeBackend{ack: syncv1.SyncAck{Accepted: 2}}
	u := New(ob, be, "edge-01", 100)

	n, err := u.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}
	if n != 2 {
		t.Errorf("ожидалось 2 отправленных, получено %d", n)
	}
	if be.called != 1 {
		t.Errorf("Backend должен вызваться 1 раз, вызван %d", be.called)
	}
	if be.got.ClientID != "edge-01" || be.got.Version != syncv1.Version {
		t.Errorf("неверная шапка пакета: %+v", be.got)
	}
	if len(ob.marked) != 2 {
		t.Errorf("ожидалось 2 помеченных, получено %d", len(ob.marked))
	}
}

func TestSyncOnce_BackendError_NoMark(t *testing.T) {
	ob := &fakeOutbox{entries: []OutboxEntry{
		{ID: 1, Payload: tickPayload(t, 24)},
	}}
	be := &fakeBackend{err: errors.New("network down")}
	u := New(ob, be, "edge-01", 100)

	_, err := u.SyncOnce(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка отправки")
	}
	if len(ob.marked) != 0 {
		t.Errorf("при ошибке Backend записи НЕ должны помечаться отправленными, помечено %d", len(ob.marked))
	}
}

func TestSyncOnce_BadPayload_Skipped(t *testing.T) {
	ob := &fakeOutbox{entries: []OutboxEntry{
		{ID: 1, Payload: "{not json"},
		{ID: 2, Payload: tickPayload(t, 24)},
	}}
	be := &fakeBackend{ack: syncv1.SyncAck{Accepted: 1}}
	u := New(ob, be, "edge-01", 100)

	n, err := u.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}
	if n != 1 {
		t.Errorf("ожидался 1 валидный тик, получено %d", n)
	}
	// Обе записи (битая + валидная) должны быть помечены.
	if len(ob.marked) != 2 {
		t.Errorf("ожидалось 2 помеченных (битая + валидная), получено %d", len(ob.marked))
	}
}

func TestSyncOnce_AllBadPayload(t *testing.T) {
	ob := &fakeOutbox{entries: []OutboxEntry{
		{ID: 1, Payload: "{not json"},
	}}
	be := &fakeBackend{}
	u := New(ob, be, "edge-01", 100)

	n, err := u.SyncOnce(context.Background())
	if err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}
	if n != 0 {
		t.Errorf("ожидалось 0 валидных, получено %d", n)
	}
	if be.called != 0 {
		t.Errorf("Backend не должен вызываться, если все записи битые")
	}
	if len(ob.marked) != 1 {
		t.Errorf("битая запись должна быть помечена, помечено %d", len(ob.marked))
	}
}
