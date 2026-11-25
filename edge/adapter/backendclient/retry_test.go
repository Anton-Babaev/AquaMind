package backendclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Anton-Babaev/AquaMind/contract/syncv1"
)

// newFastClient — клиент с минимальным backoff, чтобы тесты не тормозили.
func newFastClient(url string) *Client {
	c := New(url, time.Second)
	c.backoff = time.Millisecond
	return c
}

func TestSync_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(syncv1.SyncAck{Accepted: 1})
	}))
	t.Cleanup(srv.Close)

	ack, err := newFastClient(srv.URL).Sync(context.Background(), testBatch())
	if err != nil {
		t.Fatalf("Sync() error = %v, want success after retries", err)
	}
	if ack.Accepted != 1 {
		t.Errorf("ack.Accepted = %d, want 1", ack.Accepted)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3 (2 fail + 1 success)", got)
	}
}

func TestSync_NoRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	if _, err := newFastClient(srv.URL).Sync(context.Background(), testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want error for 400")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1 (4xx не повторяем)", got)
	}
}

func TestSync_ExhaustsRetriesOn5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := newFastClient(srv.URL)
	if _, err := c.Sync(context.Background(), testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want error after exhausting retries")
	}
	want := int32(c.retries + 1)
	if got := atomic.LoadInt32(&calls); got != want {
		t.Errorf("calls = %d, want %d", got, want)
	}
}

func TestSync_RetryStopsOnContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := newFastClient(srv.URL)
	c.backoff = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, err := c.Sync(ctx, testBatch()); err == nil {
		t.Fatal("Sync() error = nil, want context error")
	}
}
