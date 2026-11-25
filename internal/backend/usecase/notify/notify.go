// Package notify реализует логику уведомлений об алертах.
//
// Задача: Edge шлёт алерты в каждом тике, пока они активны. Backend не должен
// слать сообщение на каждый тик. Логика:
//   - raised  → отправить один раз (с учётом cooldown на повторный raised);
//   - cleared → отправить «восстановлено» (если включено).
//
// Состояние «уже отправлено» хранится в памяти процесса (для MVP). При
// перезапуске backend состояние сбрасывается — это допустимо: худший случай
// одно лишнее сообщение.
package notify

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/backend/domain"
)

// Sender — канал доставки сообщения (Telegram, SMS, ...).
type Sender interface {
	// Send отправляет текст в канал фермера.
	Send(ctx context.Context, text string) error
	// Name — имя канала для логов ("telegram", "sms").
	Name() string
}

// Usecase — логика уведомлений с антиспамом.
type Usecase struct {
	sender        Sender
	cooldown      time.Duration
	notifyCleared bool

	mu   sync.Mutex
	last map[string]time.Time // ключ: clientID|alertType → время последней отправки raised
}

// New создаёт usecase уведомлений.
func New(sender Sender, cooldown time.Duration, notifyCleared bool) *Usecase {
	return &Usecase{
		sender:        sender,
		cooldown:      cooldown,
		notifyCleared: notifyCleared,
		last:          make(map[string]time.Time),
	}
}

// Notify обрабатывает алерты и отправляет нужные сообщения.
func (u *Usecase) Notify(ctx context.Context, clientID string, alerts []domain.Alert) error {
	var firstErr error
	for _, a := range alerts {
		if err := u.handleOne(ctx, clientID, a); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (u *Usecase) handleOne(ctx context.Context, clientID string, a domain.Alert) error {
	switch a.State {
	case domain.AlertRaised:
		return u.handleRaised(ctx, clientID, a)
	case domain.AlertCleared:
		return u.handleCleared(ctx, clientID, a)
	default:
		return nil
	}
}

func (u *Usecase) handleRaised(ctx context.Context, clientID string, a domain.Alert) error {
	key := clientID + "|" + string(a.Type)

	u.mu.Lock()
	lastSent, ok := u.last[key]
	now := time.Now()
	if ok && now.Sub(lastSent) < u.cooldown {
		u.mu.Unlock()
		return nil // ещё в пределах cooldown — не спамим
	}
	u.last[key] = now
	u.mu.Unlock()

	text := formatRaised(a)
	if err := u.sender.Send(ctx, text); err != nil {
		return fmt.Errorf("send raised %s: %w", a.Type, err)
	}
	log.Printf("notify[%s]: sent raised %s for %s", u.sender.Name(), a.Type, clientID)
	return nil
}

func (u *Usecase) handleCleared(ctx context.Context, clientID string, a domain.Alert) error {
	key := clientID + "|" + string(a.Type)

	u.mu.Lock()
	_, wasRaised := u.last[key]
	delete(u.last, key) // сбрасываем состояние: следующий raised снова уведомит
	u.mu.Unlock()

	if !u.notifyCleared || !wasRaised {
		return nil
	}

	text := formatCleared(a)
	if err := u.sender.Send(ctx, text); err != nil {
		return fmt.Errorf("send cleared %s: %w", a.Type, err)
	}
	log.Printf("notify[%s]: sent cleared %s for %s", u.sender.Name(), a.Type, clientID)
	return nil
}

// formatRaised формирует текст сообщения о тревоге.
func formatRaised(a domain.Alert) string {
	icon := "⚠️"
	if a.Severity == domain.SeverityCritical {
		icon = "🚨"
	}
	return fmt.Sprintf("%s ТРЕВОГА: %s\n%s\nВремя: %s",
		icon, a.Type, a.Message, a.At.Format("2006-01-02 15:04:05"))
}

// formatCleared формирует текст сообщения о восстановлении.
func formatCleared(a domain.Alert) string {
	return fmt.Sprintf("✅ Восстановлено: %s\n%s\nВремя: %s",
		a.Type, a.Message, a.At.Format("2006-01-02 15:04:05"))
}
