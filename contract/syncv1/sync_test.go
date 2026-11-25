package syncv1

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestSyncBatch_RoundTrip — сериализация и обратная десериализация пакета
// не теряют данные. Это гарантия совместимости Edge↔Backend.
func TestSyncBatch_RoundTrip(t *testing.T) {
	at := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	orig := SyncBatch{
		Version:  Version,
		ClientID: "pi-001",
		Ticks: []TickDTO{
			{
				Reading: ReadingDTO{
					TempC: 24.5, FlowLPM: 12.0, O2MgL: 7.1, PH: 7.2, At: at,
				},
				Actuator: ActuatorDTO{Heating: true, Normal: true},
				Alerts: []AlertDTO{
					{Type: "temp_high", Severity: "critical", Message: "высокая t°", At: at, State: "raised"},
				},
			},
		},
	}

	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SyncBatch
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != orig.Version || got.ClientID != orig.ClientID {
		t.Fatalf("шапка не совпала: %+v", got)
	}
	if len(got.Ticks) != 1 {
		t.Fatalf("ожидался 1 тик, получено %d", len(got.Ticks))
	}
	tk := got.Ticks[0]
	if tk.Reading.TempC != 24.5 || !tk.Reading.At.Equal(at) {
		t.Fatalf("показание не совпало: %+v", tk.Reading)
	}
	if !tk.Actuator.Heating || !tk.Actuator.Normal {
		t.Fatalf("выходы не совпали: %+v", tk.Actuator)
	}
	if len(tk.Alerts) != 1 || tk.Alerts[0].Type != "temp_high" {
		t.Fatalf("алерты не совпали: %+v", tk.Alerts)
	}
}

// TestSyncBatch_JSONFieldNames — фиксируем имена JSON-полей.
// Если кто-то переименует тег, тест упадёт — это защита контракта.
func TestSyncBatch_JSONFieldNames(t *testing.T) {
	raw, err := json.Marshal(SyncBatch{Version: 1, ClientID: "x"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(raw)
	for _, want := range []string{`"version"`, `"client_id"`, `"ticks"`} {
		if !strings.Contains(s, want) {
			t.Errorf("в JSON нет поля %s: %s", want, s)
		}
	}
}

// TestSyncAck_JSONFieldNames — имена полей ответа.
func TestSyncAck_JSONFieldNames(t *testing.T) {
	raw, err := json.Marshal(SyncAck{Accepted: 3, Duplicates: 1})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(raw)
	for _, want := range []string{`"accepted"`, `"duplicates"`} {
		if !strings.Contains(s, want) {
			t.Errorf("в JSON нет поля %s: %s", want, s)
		}
	}
}
