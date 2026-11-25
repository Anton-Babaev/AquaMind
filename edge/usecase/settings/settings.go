// Package settings реализует сценарий управления порогами правил.
//
// Usecase — единственное место, где пороги валидируются перед сохранением.
// Delivery (HTTP) и хранилище (SQLite) не принимают решений о корректности:
// HTTP лишь передаёт данные, SQLite лишь хранит. Это гарантирует, что
// некорректные пороги не попадут в БД ни через API, ни иным путём.
package settings

import (
	"context"
	"fmt"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// Usecase — сценарий управления порогами.
type Usecase struct {
	store Store
}

// New создаёт Usecase.
func New(store Store) *Usecase {
	return &Usecase{store: store}
}

// Load возвращает текущие пороги.
func (u *Usecase) Load(ctx context.Context) (domain.Thresholds, error) {
	return u.store.LoadThresholds(ctx)
}

// Save валидирует и сохраняет пороги.
// Возвращает человекочитаемую ошибку валидации (её видит пользователь в UI).
func (u *Usecase) Save(ctx context.Context, t domain.Thresholds) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("некорректные пороги: %w", err)
	}
	return u.store.SaveThresholds(ctx, t)
}

// LoadThresholds — алиас Load для совместимости с интерфейсом delivery/http.
func (u *Usecase) LoadThresholds(ctx context.Context) (domain.Thresholds, error) {
	return u.Load(ctx)
}

// SaveThresholds — алиас Save для совместимости с интерфейсом delivery/http.
func (u *Usecase) SaveThresholds(ctx context.Context, t domain.Thresholds) error {
	return u.Save(ctx, t)
}
