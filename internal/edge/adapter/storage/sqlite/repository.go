package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Anton-Babaev/AquaMind/internal/contract/syncv1"
	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Repository — SQLite-реализация портов control.Store и settings.Store.
type Repository struct {
	db *sql.DB
}

// Open открывает БД, включает WAL и применяет миграции.
func Open(path string) (*Repository, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite: один писатель, избегаем блокировок

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	r := &Repository{db: db}
	if err := r.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

// Close закрывает БД.
func (r *Repository) Close() error { return r.db.Close() }

// migrate применяет SQL-миграции из embed.
func (r *Repository) migrate() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", e.Name(), err)
		}
		if _, err := r.db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", e.Name(), err)
		}
	}
	return nil
}

// SaveTick сохраняет показание, состояние выходов, алерты и запись outbox
// одной транзакцией (правило 12).
func (r *Repository) SaveTick(ctx context.Context, reading domain.SensorReading, st domain.ActuatorState, alerts []domain.Alert) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	atMs := reading.At.UnixMilli()

	// readings
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO readings (at_ms, temp_c, flow_lpm, o2_mgl, ph, heating, alarm, normal, safe_mode)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		atMs, reading.TempC, reading.FlowLPM, reading.O2MgL, reading.PH,
		boolToInt(st.Heating), boolToInt(st.Alarm), boolToInt(st.Normal), boolToInt(st.SafeMode),
	); err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}

	// alerts
	for _, a := range alerts {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO alerts (at_ms, type, severity, message, state) VALUES (?, ?, ?, ?, ?)`,
			a.At.UnixMilli(), string(a.Type), string(a.Severity), a.Message, string(a.State),
		); err != nil {
			return fmt.Errorf("insert alert: %w", err)
		}
	}

	// outbox: сериализуем тик в JSON.
	tick := syncv1.TickDTO{
		Reading: syncv1.ReadingDTO{
			TempC: reading.TempC, FlowLPM: reading.FlowLPM, O2MgL: reading.O2MgL, PH: reading.PH, At: reading.At,
		},
		Actuator: syncv1.ActuatorDTO{
			Heating: st.Heating, Alarm: st.Alarm, Normal: st.Normal, SafeMode: st.SafeMode,
		},
		Alerts: toAlertDTOs(alerts),
	}
	payload, err := json.Marshal(tick)
	if err != nil {
		return fmt.Errorf("marshal tick: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO outbox (at_ms, payload, sent) VALUES (?, ?, 0)`,
		atMs, string(payload),
	); err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	return tx.Commit()
}

// LoadThresholds читает пороги из settings.
func (r *Repository) LoadThresholds(ctx context.Context) (domain.Thresholds, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return domain.Thresholds{}, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return domain.Thresholds{}, err
		}
		m[k] = v
	}
	if err := rows.Err(); err != nil {
		return domain.Thresholds{}, err
	}

	// Если настроек нет — возвращаем значения по умолчанию.
	return thresholdsFromMap(m), nil
}

// SaveThresholds сохраняет пороги в settings.
func (r *Repository) SaveThresholds(ctx context.Context, t domain.Thresholds) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for k, v := range thresholdsToMap(t) {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			k, v,
		); err != nil {
			return fmt.Errorf("upsert setting %s: %w", k, err)
		}
	}
	return tx.Commit()
}

// RecentAlerts возвращает последние limit событий алертов (raised/cleared),
// отсортированные по времени по убыванию (свежие первыми).
func (r *Repository) RecentAlerts(ctx context.Context, limit int) ([]domain.Alert, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT at_ms, type, severity, message, state FROM alerts ORDER BY at_ms DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()

	var out []domain.Alert
	for rows.Next() {
		var (
			atMs                      int64
			typ, severity, msg, state string
		)
		if err := rows.Scan(&atMs, &typ, &severity, &msg, &state); err != nil {
			return nil, err
		}
		out = append(out, domain.Alert{
			Type:     domain.AlertType(typ),
			Severity: domain.AlertSeverity(severity),
			Message:  msg,
			At:       time.UnixMilli(atMs).UTC(),
			State:    domain.AlertState(state),
		})
	}
	return out, rows.Err()
}

// PendingOutbox возвращает до limit неотправленных записей outbox.
func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]OutboxEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, at_ms, payload FROM outbox WHERE sent = 0 ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query outbox: %w", err)
	}
	defer rows.Close()

	var out []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var atMs int64
		if err := rows.Scan(&e.ID, &atMs, &e.Payload); err != nil {
			return nil, err
		}
		e.At = time.UnixMilli(atMs).UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkSent помечает записи outbox как отправленные.
func (r *Repository) MarkSent(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().UnixMilli()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE outbox SET sent = 1, sent_at_ms = ? WHERE id = ?`, now, id,
		); err != nil {
			return fmt.Errorf("mark sent %d: %w", id, err)
		}
	}
	return tx.Commit()
}

// OutboxEntry — запись outbox.
type OutboxEntry struct {
	ID      int64
	At      time.Time
	Payload string
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func toAlertDTOs(alerts []domain.Alert) []syncv1.AlertDTO {
	if len(alerts) == 0 {
		return nil
	}
	out := make([]syncv1.AlertDTO, len(alerts))
	for i, a := range alerts {
		out[i] = syncv1.AlertDTO{
			Type: string(a.Type), Severity: string(a.Severity),
			Message: a.Message, At: a.At, State: string(a.State),
		}
	}
	return out
}
