package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/backend/domain"
	"github.com/Anton-Babaev/AquaMind/contract/syncv1"
)

// --- фейки ---

type fakeIngest struct {
	ack syncv1.SyncAck
	err error
	got syncv1.SyncBatch
}

func (f *fakeIngest) Handle(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error) {
	f.got = batch
	return f.ack, f.err
}

type fakePinger struct{ err error }

func (f *fakePinger) Ping(ctx context.Context) error { return f.err }

type fakeQuery struct {
	clients    []domain.Client
	clientsErr error

	tick    domain.Tick
	tickOK  bool
	tickErr error

	ticks    []domain.Tick
	ticksErr error

	alerts    []domain.Alert
	alertsErr error
}

func (f *fakeQuery) Clients(ctx context.Context) ([]domain.Client, error) { return f.clients, f.clientsErr }
func (f *fakeQuery) State(ctx context.Context, id string) (domain.Tick, bool, error) {
	return f.tick, f.tickOK, f.tickErr
}
func (f *fakeQuery) Ticks(ctx context.Context, id string, from, to time.Time, limit int) ([]domain.Tick, error) {
	return f.ticks, f.ticksErr
}
func (f *fakeQuery) Alerts(ctx context.Context, id string, limit int) ([]domain.Alert, error) {
	return f.alerts, f.alertsErr
}

func newTestServer(ing Ingest, p Pinger, q Query) http.Handler {
	return NewServer(ing, p, q).Handler()
}

// --- тесты ---

func TestHandleSync_OK(t *testing.T) {
	ing := &fakeIngest{ack: syncv1.SyncAck{Accepted: 2, Duplicates: 1}}
	h := newTestServer(ing, &fakePinger{}, &fakeQuery{})

	body := `{"version":1,"client_id":"edge-01","ticks":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var ack syncv1.SyncAck
	if err := json.NewDecoder(rec.Body).Decode(&ack); err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if ack.Accepted != 2 || ack.Duplicates != 1 {
		t.Errorf("ack = %+v, want accepted=2 duplicates=1", ack)
	}
	if ing.got.ClientID != "edge-01" {
		t.Errorf("ingest got client_id = %q, want edge-01", ing.got.ClientID)
	}
}

func TestHandleSync_MethodNotAllowed(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHandleSync_BadJSON(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSync_IngestError(t *testing.T) {
	ing := &fakeIngest{err: errors.New("db down")}
	h := newTestServer(ing, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader(`{"client_id":"x"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestHandleHealth_OK(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestHandleHealth_DBDown(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{err: errors.New("down")}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestHandleClients(t *testing.T) {
	now := time.Now().UTC()
	q := &fakeQuery{clients: []domain.Client{{ClientID: "edge-01", FirstSeen: now, LastSeen: now}}}
	h := newTestServer(&fakeIngest{}, &fakePinger{}, q)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []clientDTO
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].ClientID != "edge-01" {
		t.Errorf("clients = %+v, want one edge-01", got)
	}
}

func TestHandleState_MissingClientID(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleState_NotFound(t *testing.T) {
	q := &fakeQuery{tickOK: false}
	h := newTestServer(&fakeIngest{}, &fakePinger{}, q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/state?client_id=edge-01", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleState_OK(t *testing.T) {
	q := &fakeQuery{
		tick:   domain.Tick{Reading: domain.Reading{TempC: 24.5}, Actuator: domain.ActuatorState{Normal: true}},
		tickOK: true,
	}
	h := newTestServer(&fakeIngest{}, &fakePinger{}, q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/state?client_id=edge-01", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got tickDTO
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Reading.TempC != 24.5 || !got.Actuator.Normal {
		t.Errorf("state = %+v, want temp 24.5 normal=true", got)
	}
}

func TestHandleTicks_MissingClientID(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ticks", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAlerts_MissingClientID(t *testing.T) {
	h := newTestServer(&fakeIngest{}, &fakePinger{}, &fakeQuery{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestParseMillis(t *testing.T) {
	if !parseMillis("").IsZero() {
		t.Error("parseMillis(\"\") should be zero")
	}
	if !parseMillis("abc").IsZero() {
		t.Error("parseMillis(\"abc\") should be zero")
	}
	if !parseMillis("-5").IsZero() {
		t.Error("parseMillis(\"-5\") should be zero")
	}
	got := parseMillis("1700000000000")
	if got.UnixMilli() != 1700000000000 {
		t.Errorf("parseMillis = %v, want 1700000000000", got.UnixMilli())
	}
}

func TestParseInt(t *testing.T) {
	if parseInt("", 42) != 42 {
		t.Error("parseInt(\"\") should return default")
	}
	if parseInt("abc", 42) != 42 {
		t.Error("parseInt(\"abc\") should return default")
	}
	if parseInt("7", 42) != 7 {
		t.Error("parseInt(\"7\") should return 7")
	}
}
