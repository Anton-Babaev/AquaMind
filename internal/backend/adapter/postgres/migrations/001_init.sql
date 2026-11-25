-- Начальная схема Backend (PostgreSQL).
--
-- Таблицы:
--   clients — известные устройства Edge (по client_id).
--   ticks   — показания датчиков + состояние выходов (по одному на тик).
--   alerts  — события алертов (raised/cleared), привязаны к тику.
--
-- Дедупликация: уникальный индекс (client_id, at_ms) на ticks. Edge досылает
-- данные из outbox при сбоях сети, поэтому один и тот же тик может прийти
-- несколько раз — повтор отбрасывается через ON CONFLICT DO NOTHING.

CREATE TABLE IF NOT EXISTS clients (
    client_id   TEXT PRIMARY KEY,
    first_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ticks (
    id          BIGSERIAL PRIMARY KEY,
    client_id   TEXT NOT NULL REFERENCES clients(client_id),
    at_ms       BIGINT NOT NULL,          -- время измерения, Unix ms (UTC)
    temp_c      DOUBLE PRECISION NOT NULL,
    flow_lpm    DOUBLE PRECISION NOT NULL,
    o2_mgl      DOUBLE PRECISION NOT NULL,
    ph          DOUBLE PRECISION NOT NULL,
    heating     BOOLEAN NOT NULL,
    alarm       BOOLEAN NOT NULL,
    normal      BOOLEAN NOT NULL,
    safe_mode   BOOLEAN NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (client_id, at_ms)
);

CREATE INDEX IF NOT EXISTS idx_ticks_client_at ON ticks (client_id, at_ms DESC);

CREATE TABLE IF NOT EXISTS alerts (
    id          BIGSERIAL PRIMARY KEY,
    tick_id     BIGINT NOT NULL REFERENCES ticks(id) ON DELETE CASCADE,
    client_id   TEXT NOT NULL,
    at_ms       BIGINT NOT NULL,
    type        TEXT NOT NULL,
    severity    TEXT NOT NULL,
    message     TEXT NOT NULL,
    state       TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alerts_client_at ON alerts (client_id, at_ms DESC);
