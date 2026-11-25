package backendclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Anton-Babaev/AquaMind/contract/syncv1"
)

// Client — HTTP-клиент Backend.
type Client struct {
	baseURL string
	http    *http.Client
	// retries — сколько раз повторять запрос при временных сбоях (сетевые
	// ошибки, 5xx). 0 — без повторов. 4xx не повторяем (это ошибка запроса).
	retries int
	// backoff — базовая пауза между попытками (экспоненциально растёт).
	backoff time.Duration
}

// New создаёт клиент с разумными ретраями по умолчанию.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
		retries: 3,
		backoff: 200 * time.Millisecond,
	}
}

// Sync отправляет пакет на Backend и возвращает подтверждение.
// При временных сбоях повторяет запрос с экспоненциальной паузой.
func (c *Client) Sync(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error) {
	body, err := json.Marshal(batch)
	if err != nil {
		return syncv1.SyncAck{}, fmt.Errorf("marshal batch: %w", err)
	}

	var lastErr error
	attempts := c.retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			// Экспоненциальная пауза: backoff, 2*backoff, 4*backoff...
			delay := c.backoff << (attempt - 1)
			select {
			case <-ctx.Done():
				return syncv1.SyncAck{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		ack, retryable, err := c.doOnce(ctx, body)
		if err == nil {
			return ack, nil
		}
		lastErr = err
		if !retryable {
			return syncv1.SyncAck{}, err
		}
	}
	return syncv1.SyncAck{}, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

// doOnce выполняет одну попытку запроса.
// Возвращает retryable=true, если ошибку имеет смысл повторить.
func (c *Client) doOnce(ctx context.Context, body []byte) (ack syncv1.SyncAck, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/sync", bytes.NewReader(body))
	if err != nil {
		return syncv1.SyncAck{}, false, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		// Сетевая ошибка/таймаут — временная, повторяем (если ctx не отменён).
		if ctx.Err() != nil {
			return syncv1.SyncAck{}, false, ctx.Err()
		}
		return syncv1.SyncAck{}, true, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 5xx — временная ошибка сервера, повторяем. 4xx — нет.
		retryable := resp.StatusCode >= 500
		return syncv1.SyncAck{}, retryable, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		// Битый ответ — повтор не поможет.
		return syncv1.SyncAck{}, false, fmt.Errorf("decode ack: %w", err)
	}
	return ack, false, nil
}
