// Package ingest реализует приём пакета синхронизации от Edge.
//
// Сценарий:
//  1. Преобразовать DTO (syncv1) в доменные тики.
//  2. Сохранить в хранилище с дедупликацией (повторы от outbox отбрасываются).
//  3. Передать алерты из ПРИНЯТЫХ тиков в Notifier (уведомления).
//  4. Вернуть SyncAck (accepted/duplicates).
package ingest

import (
	"context"
	"fmt"

	"github.com/Anton-Babaev/AquaMind/internal/backend/domain"
	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
)

// Usecase — приём пакетов синхронизации.
type Usecase struct {
	store    Store
	notifier Notifier
}

// New создаёт usecase приёма.
func New(store Store, notifier Notifier) *Usecase {
	return &Usecase{store: store, notifier: notifier}
}

// Handle принимает пакет от Edge и возвращает подтверждение.
func (u *Usecase) Handle(ctx context.Context, batch syncv1.SyncBatch) (syncv1.SyncAck, error) {
	if batch.ClientID == "" {
		return syncv1.SyncAck{}, fmt.Errorf("empty client_id")
	}

	ticks := toDomainTicks(batch.Ticks)

	accepted, duplicates, err := u.store.SaveBatch(ctx, batch.ClientID, ticks)
	if err != nil {
		return syncv1.SyncAck{}, fmt.Errorf("save batch: %w", err)
	}

	// Уведомления отправляем только по алертам из принятых (новых) тиков,
	// чтобы не дублировать сообщения при повторной досылке из outbox.
	if u.notifier != nil && accepted > 0 {
		var alerts []domain.Alert
		for _, t := range ticks {
			alerts = append(alerts, t.Alerts...)
		}
		if len(alerts) > 0 {
			if err := u.notifier.Notify(ctx, batch.ClientID, alerts); err != nil {
				// Ошибка уведомления не должна ломать приём данных:
				// данные уже сохранены, алерт можно отправить позже.
				return syncv1.SyncAck{Accepted: accepted, Duplicates: duplicates},
					fmt.Errorf("notify: %w", err)
			}
		}
	}

	return syncv1.SyncAck{Accepted: accepted, Duplicates: duplicates}, nil
}

// toDomainTicks преобразует DTO в доменные тики.
func toDomainTicks(dtos []syncv1.TickDTO) []domain.Tick {
	out := make([]domain.Tick, 0, len(dtos))
	for _, d := range dtos {
		t := domain.Tick{
			Reading: domain.Reading{
				TempC:   d.Reading.TempC,
				FlowLPM: d.Reading.FlowLPM,
				O2MgL:   d.Reading.O2MgL,
				PH:      d.Reading.PH,
				At:      d.Reading.At.UTC(),
			},
			Actuator: domain.ActuatorState{
				Heating:  d.Actuator.Heating,
				Alarm:    d.Actuator.Alarm,
				Normal:   d.Actuator.Normal,
				SafeMode: d.Actuator.SafeMode,
			},
		}
		for _, a := range d.Alerts {
			t.Alerts = append(t.Alerts, domain.Alert{
				Type:     domain.AlertType(a.Type),
				Severity: domain.AlertSeverity(a.Severity),
				Message:  a.Message,
				At:       a.At.UTC(),
				State:    domain.AlertState(a.State),
			})
		}
		out = append(out, t)
	}
	return out
}
