package settings

import (
	"context"
	"testing"

	"github.com/Anton-Babaev/AquaMind/edge/domain"
)

// fakeStore — поддельное хранилище порогов.
type fakeStore struct {
	saved   domain.Thresholds
	saveCnt int
	load    domain.Thresholds
}

func (f *fakeStore) LoadThresholds(context.Context) (domain.Thresholds, error) {
	return f.load, nil
}

func (f *fakeStore) SaveThresholds(_ context.Context, t domain.Thresholds) error {
	f.saved = t
	f.saveCnt++
	return nil
}

func validThresholds() domain.Thresholds {
	return domain.Thresholds{
		TempLow:           22.0,
		TempLowHysteresis: 1.0,
		TempCriticalLow:   18.0,
		TempCriticalHigh:  30.0,
		O2Low:             5.0,
		O2High:            9.0,
		O2LowHysteresis:   0.5,
		FlowMin:           5.0,
		PHLow:             6.5,
		PHHigh:            8.5,
	}
}

func TestSave_Valid(t *testing.T) {
	store := &fakeStore{}
	u := New(store)

	if err := u.Save(context.Background(), validThresholds()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if store.saveCnt != 1 {
		t.Errorf("ожидалось 1 сохранение, получено %d", store.saveCnt)
	}
}

func TestSave_Invalid(t *testing.T) {
	store := &fakeStore{}
	u := New(store)

	bad := validThresholds()
	bad.O2Low = 10.0 // больше O2High — невалидно

	if err := u.Save(context.Background(), bad); err == nil {
		t.Fatal("ожидалась ошибка валидации, получено nil")
	}
	if store.saveCnt != 0 {
		t.Errorf("невалидные пороги не должны сохраняться, но сохранение было")
	}
}

func TestLoad(t *testing.T) {
	want := validThresholds()
	store := &fakeStore{load: want}
	u := New(store)

	got, err := u.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}
