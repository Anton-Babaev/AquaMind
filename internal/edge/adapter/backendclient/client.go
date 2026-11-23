package backendclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
)

// Client — HTTP-клиент Backend.
type Client struct {
	baseURL string
	http    *http.Client
}

// New создаёт клиент.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// Sync отправляет пакет на Backend и возвращает подтверждение.
func (c *Client) Sync(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error) {
	body, err := json.Marshal(batch)
	if err != nil {
		return syncv1.SyncAck{}, fmt.Errorf("marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sync", bytes.NewReader(body))
	if err != nil {
		return syncv1.SyncAck{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return syncv1.SyncAck{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return syncv1.SyncAck{}, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var ack syncv1.SyncAck
	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		return syncv1.SyncAck{}, fmt.Errorf("decode ack: %w", err)
	}
	return ack, nil
}
