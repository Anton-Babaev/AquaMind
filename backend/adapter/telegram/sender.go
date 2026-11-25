// Package telegram — адаптер отправки сообщений в Telegram Bot API.
//
// Использует метод sendMessage:
//
//	POST https://api.telegram.org/bot<token>/sendMessage
//	{"chat_id": "...", "text": "...", "parse_mode": "HTML"}
//
// Токен и chat_id задаются в конфиге (на хостинге — через env).
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Sender — Telegram-реализация notify.Sender.
type Sender struct {
	token   string
	chatID  string
	client  *http.Client
	baseURL string // для тестов можно подменить на httptest-сервер
}

// New создаёт Telegram-отправитель.
func New(token, chatID string, timeout time.Duration) *Sender {
	return &Sender{
		token:   token,
		chatID:  chatID,
		client:  &http.Client{Timeout: timeout},
		baseURL: "https://api.telegram.org",
	}
}

// Name возвращает имя канала.
func (s *Sender) Name() string { return "telegram" }

// sendMessageRequest — тело запроса sendMessage.
type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// apiResponse — ответ Telegram API.
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// Send отправляет текст в чат фермера.
func (s *Sender) Send(ctx context.Context, text string) error {
	body, err := json.Marshal(sendMessageRequest{
		ChatID:    s.chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", s.baseURL, s.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, string(respBody))
	}

	var ar apiResponse
	if err := json.Unmarshal(respBody, &ar); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !ar.OK {
		return fmt.Errorf("telegram error: %s", ar.Description)
	}
	return nil
}
