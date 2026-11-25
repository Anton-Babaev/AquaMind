package postgres

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Anton-Babaev/AquaMind/internal/backend/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Repository — PostgreSQL-реализация хранилища Backend.
type Repository struct {
	pool *pgxpool.Pool
}

// Open открывает пул соединений и применяет миграции.
func Open(ctx context.Context, dsn string, maxConns int, connectTimeout time.Duration) (*Repository, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = int32(maxConns)
	}
	if connectTimeout > 0 {
		cfg.ConnConfig.ConnectTimeout = connectTimeout
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	r := &Repository{pool: pool}
	if err := r.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return r, nil
}

// Close закрывает пул соединений.
func (r *Repository) Close() { r.pool.Close() }

// migrate применяет SQL-миграции из embed (идемпотентно: CREATE ... IF NOT EXISTS).
func (r *Repository) migrate(ctx context.Context) error {
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
		if _, err := r.pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", e.Name(), err)
		}
	}
	return nil
}

// SaveBatch сохраняет пакет тиков от одного клиента и возвращает количество
// принятых (новых) и отброшенных (дубликаты) тиков.
//
// Дедупликация — по уникальному индексу (client_id, at_ms): повторный тик
// отбрасывается через ON CONFLICT DO NOTHING. Всё в одной транзакции.
func (r *Repository) SaveBatch(ctx context.Context, clientID string, ticks []domain.Tick) (accepted, duplicates int, err error) {
	if len(ticks) == 0 {
		return 0, 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Регистрируем/обновляем клиента.
	if _, err := tx.Exec(ctx,
		`INSERT INTO clients (client_id, last_seen) VALUES ($1, now())
		 ON CONFLICT (client_id) DO UPDATE SET last_seen = now()`,
		clientID,
	); err != nil {
		return 0, 0, fmt.Errorf("upsert client: %w", err)
	}

	for _, t := range ticks {
		atMs := t.Reading.At.UnixMilli()

		var tickID int64
		err := tx.QueryRow(ctx,
			`INSERT INTO ticks
			   (client_id, at_ms, temp_c, flow_lpm, o2_mgl, ph, heating, alarm, normal, safe_mode)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 ON CONFLICT (client_id, at_ms) DO NOTHING
			 RETURNING id`,
			clientID, atMs, t.Reading.TempC, t.Reading.FlowLPM, t.Reading.O2MgL, t.Reading.PH,
			t.Actuator.Heating, t.Actuator.Alarm, t.Actuator.Normal, t.Actuator.SafeMode,
		).Scan(&tickID)
		if err != nil {
			// pgx.ErrNoRows → конфликт (дубликат), тик уже был сохранён.
			if isNoRows(err) {
				duplicates++
				continue
			}
			return 0, 0, fmt.Errorf("insert tick: %w", err)
		}
		accepted++

		// Алерты сохраняем только для новых тиков (у дубликатов они уже есть).
		for _, a := range t.Alerts {
			if _, err := tx.Exec(ctx,
				`INSERT INTO alerts (tick_id, client_id, at_ms, type, severity, message, state)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				tickID, clientID, a.At.UnixMilli(), string(a.Type), string(a.Severity), a.Message, string(a.State),
			); err != nil {
				return 0, 0, fmt.Errorf("insert alert: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return accepted, duplicates, nil
}

// RecentTicks возвращает последние limit тиков клиента (свежие первыми).
// Используется будущим frontend-API; сейчас — для проверки/отладки.
func (r *Repository) RecentTicks(ctx context.Context, clientID string, limit int) ([]domain.Tick, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT at_ms, temp_c, flow_lpm, o2_mgl, ph, heating, alarm, normal, safe_mode
		 FROM ticks WHERE client_id = $1 ORDER BY at_ms DESC LIMIT $2`,
		clientID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query ticks: %w", err)
	}
	defer rows.Close()

	var out []domain.Tick
	for rows.Next() {
		var (
			atMs int64
			t    domain.Tick
		)
		if err := rows.Scan(&atMs, &t.Reading.TempC, &t.Reading.FlowLPM, &t.Reading.O2MgL, &t.Reading.PH,
			&t.Actuator.Heating, &t.Actuator.Alarm, &t.Actuator.Normal, &t.Actuator.SafeMode); err != nil {
			return nil, err
		}
		t.Reading.At = time.UnixMilli(atMs).UTC()
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListClients возвращает всех известных клиентов (свежие по last_seen первыми).

func (r *Repository) ListClients(ctx context.Context) ([]domain.Client, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT client_id, first_seen, last_seen FROM clients ORDER BY last_seen DESC`)
	if err != nil {
		return nil, fmt.Errorf("query clients: %w", err)
	}
	defer rows.Close()

	var out []domain.Client
	for rows.Next() {
		var c domain.Client
		if err := rows.Scan(&c.ClientID, &c.FirstSeen, &c.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LatestTick возвращает последний тик клиента. ok=false, если тиков нет.
func (r *Repository) LatestTick(ctx context.Context, clientID string) (domain.Tick, bool, error) {
	var (
		atMs int64
		t    domain.Tick
	)
	err := r.pool.QueryRow(ctx,
		`SELECT at_ms, temp_c, flow_lpm, o2_mgl, ph, heating, alarm, normal, safe_mode
		 FROM ticks WHERE client_id = $1 ORDER BY at_ms DESC LIMIT 1`,
		clientID,
	).Scan(&atMs, &t.Reading.TempC, &t.Reading.FlowLPM, &t.Reading.O2MgL, &t.Reading.PH,
		&t.Actuator.Heating, &t.Actuator.Alarm, &t.Actuator.Normal, &t.Actuator.SafeMode)
	if err != nil {
		if isNoRows(err) {
			return domain.Tick{}, false, nil
		}
		return domain.Tick{}, false, fmt.Errorf("query latest tick: %w", err)
	}
	t.Reading.At = time.UnixMilli(atMs).UTC()
	return t, true, nil
}

// TicksRange возвращает тики клиента в диапазоне [from, to] (включительно),
// свежие первыми, не более limit. Нулевые from/to означают «без ограничения».
func (r *Repository) TicksRange(ctx context.Context, clientID string, from, to time.Time, limit int) ([]domain.Tick, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx,
		`SELECT at_ms, temp_c, flow_lpm, o2_mgl, ph, heating, alarm, normal, safe_mode
		 FROM ticks
		 WHERE client_id = $1
		   AND ($2 = 0 OR at_ms >= $2)
		   AND ($3 = 0 OR at_ms <= $3)
		 ORDER BY at_ms DESC LIMIT $4`,
		clientID, millisOrZero(from), millisOrZero(to), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query ticks range: %w", err)
	}
	defer rows.Close()

	var out []domain.Tick
	for rows.Next() {
		var (
			atMs int64
			t    domain.Tick
		)
		if err := rows.Scan(&atMs, &t.Reading.TempC, &t.Reading.FlowLPM, &t.Reading.O2MgL, &t.Reading.PH,
			&t.Actuator.Heating, &t.Actuator.Alarm, &t.Actuator.Normal, &t.Actuator.SafeMode); err != nil {
			return nil, err
		}
		t.Reading.At = time.UnixMilli(atMs).UTC()
		out = append(out, t)
	}
	return out, rows.Err()
}

// Alerts возвращает последние limit событий алертов клиента (свежие первыми).
func (r *Repository) Alerts(ctx context.Context, clientID string, limit int) ([]domain.Alert, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT at_ms, type, severity, message, state
		 FROM alerts WHERE client_id = $1 ORDER BY at_ms DESC LIMIT $2`,
		clientID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()

	var out []domain.Alert
	for rows.Next() {
		var (
			atMs int64
			a    domain.Alert
		)
		if err := rows.Scan(&atMs, &a.Type, &a.Severity, &a.Message, &a.State); err != nil {
			return nil, err
		}
		a.At = time.UnixMilli(atMs).UTC()
		out = append(out, a)
	}
	return out, rows.Err()
}

// millisOrZero возвращает Unix-миллисекунды времени или 0 для нулевого времени.
func millisOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// Ping проверяет доступность базы (для /healthz).
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
