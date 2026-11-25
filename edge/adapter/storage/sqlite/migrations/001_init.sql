-- Начальная схема Edge (SQLite, WAL).
-- Время храним как Unix-миллисекунды (INTEGER) для простоты и точности.

CREATE TABLE IF NOT EXISTS readings (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    at_ms     INTEGER NOT NULL,
    temp_c    REAL    NOT NULL,
    flow_lpm  REAL    NOT NULL,
    o2_mgl    REAL    NOT NULL,
    ph        REAL    NOT NULL,
    heating   INTEGER NOT NULL,
    alarm     INTEGER NOT NULL,
    normal    INTEGER NOT NULL,
    safe_mode INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_readings_at ON readings(at_ms);

CREATE TABLE IF NOT EXISTS alerts (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    at_ms    INTEGER NOT NULL,
    type     TEXT    NOT NULL,
    severity TEXT    NOT NULL,
    message  TEXT    NOT NULL,
    state    TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alerts_at ON alerts(at_ms);

-- Outbox: тики, ожидающие отправки на Backend.
-- sent=0 — не отправлено, sent=1 — отправлено (можно чистить по retention).
CREATE TABLE IF NOT EXISTS outbox (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    at_ms     INTEGER NOT NULL,
    payload   TEXT    NOT NULL, -- JSON TickDTO
    sent      INTEGER NOT NULL DEFAULT 0,
    sent_at_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_outbox_sent ON outbox(sent, id);

-- Настройки (пороги) — key/value.
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
