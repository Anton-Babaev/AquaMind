package alert

import (
	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// Tracker — реализация порта AlertTracker.
// Держит множество активных типов алертов и выдаёт события только на переходах:
//   - тип появился впервые  → raised
//   - тип пропал            → cleared
//   - тип был и остался     → ничего (антиспам)
//
// Tracker не сохраняет данные сам — это делает ControlUsecase одной транзакцией.
type Tracker struct {
	active map[domain.AlertType]domain.Alert
}

// NewTracker создаёт пустой трекер.
func NewTracker() *Tracker {
	return &Tracker{active: make(map[domain.AlertType]domain.Alert)}
}

// Track принимает активные алерты текущего тика и возвращает события переходов.
func (t *Tracker) Track(active []domain.Alert) []domain.Alert {
	var events []domain.Alert

	// Текущее множество типов.
	current := make(map[domain.AlertType]domain.Alert, len(active))
	for _, a := range active {
		current[a.Type] = a
	}

	// Новые алерты (raised).
	for typ, a := range current {
		if _, existed := t.active[typ]; !existed {
			a.State = domain.AlertRaised
			events = append(events, a)
		}
	}

	// Снятые алерты (cleared).
	for typ, a := range t.active {
		if _, still := current[typ]; !still {
			a.State = domain.AlertCleared
			events = append(events, a)
		}
	}

	// Обновляем состояние.
	t.active = current
	return events
}

// Active возвращает текущие активные алерты (для индикации).
func (t *Tracker) Active() []domain.Alert {
	out := make([]domain.Alert, 0, len(t.active))
	for _, a := range t.active {
		out = append(out, a)
	}
	return out
}

// IsActive сообщает, активен ли сейчас алерт данного типа.
// Используется для гистерезиса (кислород держим до «порог + гистерезис»).
func (t *Tracker) IsActive(typ domain.AlertType) bool {
	_, ok := t.active[typ]
	return ok
}
