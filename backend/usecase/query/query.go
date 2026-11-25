// Package query реализует чтение данных для Frontend (без регистрации).
//
// Сценарии:
//   - список устройств (клиентов);
//   - последнее состояние устройства (для «живого» дашборда);
//   - история показаний за период (для графиков);
//   - история алертов.
package query

import (
	"context"
	"fmt"
	"time"

	"github.com/Anton-Babaev/AquaMind/backend/domain"
)

// Usecase — чтение данных Backend.
type Usecase struct {
	reader Reader
}

// New создаёт usecase чтения.
func New(reader Reader) *Usecase {
	return &Usecase{reader: reader}
}

// Clients возвращает список известных устройств.
func (u *Usecase) Clients(ctx context.Context) ([]domain.Client, error) {
	clients, err := u.reader.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	return clients, nil
}

// State возвращает последнее состояние устройства. ok=false, если данных нет.
func (u *Usecase) State(ctx context.Context, clientID string) (domain.Tick, bool, error) {
	if clientID == "" {
		return domain.Tick{}, false, fmt.Errorf("empty client_id")
	}
	tick, ok, err := u.reader.LatestTick(ctx, clientID)
	if err != nil {
		return domain.Tick{}, false, fmt.Errorf("latest tick: %w", err)
	}
	return tick, ok, nil
}

// Ticks возвращает историю показаний устройства за период [from, to].
// Нулевые from/to означают «без ограничения».
func (u *Usecase) Ticks(ctx context.Context, clientID string, from, to time.Time, limit int) ([]domain.Tick, error) {
	if clientID == "" {
		return nil, fmt.Errorf("empty client_id")
	}
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	ticks, err := u.reader.TicksRange(ctx, clientID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("ticks range: %w", err)
	}
	return ticks, nil
}

// Alerts возвращает историю алертов устройства.
func (u *Usecase) Alerts(ctx context.Context, clientID string, limit int) ([]domain.Alert, error) {
	if clientID == "" {
		return nil, fmt.Errorf("empty client_id")
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	alerts, err := u.reader.Alerts(ctx, clientID, limit)
	if err != nil {
		return nil, fmt.Errorf("alerts: %w", err)
	}
	return alerts, nil
}
