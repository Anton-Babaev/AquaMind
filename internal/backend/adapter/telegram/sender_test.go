package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSend_Success(t *testing.T) {
	var gotPath string
	var gotReq sendMessageRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	s := New("TESTTOKEN", "12345", 5*time.Second)
	s.baseURL = srv.URL

	if err := s.Send(context.Background(), "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotPath != "/botTESTTOKEN/sendMessage" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotReq.ChatID != "12345" {
		t.Fatalf("want chat_id 12345, got %s", gotReq.ChatID)
	}
	if gotReq.Text != "hello" {
		t.Fatalf("want text hello, got %s", gotReq.Text)
	}
}

func TestSend_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"description":"chat not found"}`))
	}))
	defer srv.Close()

	s := New("T", "1", 5*time.Second)
	s.baseURL = srv.URL

	err := s.Send(context.Background(), "hi")
	if err == nil {
		t.Fatal("want error for ok=false")
	}
}

func TestSend_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	s := New("T", "1", 5*time.Second)
	s.baseURL = srv.URL

	if err := s.Send(context.Background(), "hi"); err == nil {
		t.Fatal("want error for 401")
	}
}
