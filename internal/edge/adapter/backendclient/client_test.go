package backendclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
)

// testBatch — небольшой пакет для тестов.
func testBatch() syncv1.SyncBatch {
	return syncv1.SyncBatch{
		Version:  syncv1.Version,
		ClientID: "edge-01",
		Ticks: []syncv1.TickDTO{
			{
				Reading:  syncv1.ReadingDTO{TempC: 24.1, FlowLPM: 12, O2MgL: 7.2, PH: 7.2, At: time.Now().UTC()},
				Actuator: syncv1.ActuatorDTO{Normal: true},
			},
		},
	}
}

func TestSync_Success(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBatch syncv1.SyncBatch

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sync", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBatch); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(syncv1.SyncAck{Accepted: 1, Duplicates: 0})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := New(srv.URL, time.Second)
	ack, err := client.Sync(context.Background(), testBatch())
	if err != nil {
		t.Fatalf("Sync() returned unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/sync" {
		t.Errorf("path = %q, want /api/v1/sync", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBatch.ClientID != "edge-01" || len(gotBatch.Ticks) != 1 {
		t.Errorf("server got batch = %+v, want client_id=edge-01 with 1 tick", gotBatch)
	}
	if ack.Accepted != 1 || ack.Duplicates != 0 {
		t.Errorf("ack = %+v, want accepted=1 duplicates=0", ack)
	}
}

func TestSync_Non2xxStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, time.Second)
	if _, err := client.Sync(context.Background(), testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want non-nil for 500 status")
	}
}

func TestSync_BadAckJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, time.Second)
	if _, err := client.Sync(context.Background(), testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want non-nil for invalid ack JSON")
	}
}

func TestSync_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(syncv1.SyncAck{})
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := New(srv.URL, time.Second)
	if _, err := client.Sync(ctx, testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want non-nil for cancelled context")
	}
}

func TestSync_Timeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // держим запрос, пока тест не отпустит
		_ = json.NewEncoder(w).Encode(syncv1.SyncAck{})
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	client := New(srv.URL, 50*time.Millisecond)
	if _, err := client.Sync(context.Background(), testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want timeout error")
	}
}

func TestSync_ConnectionRefused(t *testing.T) {
	// Сервер поднимаем и сразу закрываем — порт недоступен.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	client := New(url, time.Second)
	if _, err := client.Sync(context.Background(), testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want connection error")
	}
}
