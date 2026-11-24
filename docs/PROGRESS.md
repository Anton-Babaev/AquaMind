# AquaMind — карта проекта и статус

> Этот файл — точка входа для продолжения работы. Прочитай его первым.
> Дипломный проект: автоматизация рыбной фермы (УЗВ) на Raspberry Pi 3B+.
> Работают два человека. **Наша зона — Edge-контроллер на Raspberry Pi (Go).**
> Backend (PostgreSQL, Telegram) — зона второго разработчика (Роман).

## Где мы сейчас (для продолжения в новом чате)

- **Последний коммит:** `f7ced49` — «docs(progress): блок "Где мы сейчас"».
  В рабочем дереве есть **незакоммиченные** изменения (см. ниже).
- **Готово:** Этапы 1–4 (домен+политики, Control+Poller+console, SQLite+Sync+
  Backend-заглушка, GPIO на Pi + README + инструкция по Pi).
- **Добавлено после Этапа 4 (НЕ закоммичено):**
  - Тесты `backendclient` (`client_test.go`, 6 кейсов на `httptest`: успех,
    500, битый JSON, отмена ctx, таймаут, отказ соединения).
  - Тесты `sqlite` (`repository_test.go`, 8 кейсов на `t.TempDir()`: миграции,
    SaveTick+outbox, MarkSent, лимит, round-trip порогов, upsert, отмена ctx).
  - `Makefile`: цели `test-race` (`go test -race ./...`) и `cover`
    (`coverage.out` + `coverage.html`); `clean` чистит coverage-файлы.
  - Папка `Lessons/` (материалы куратора по Go) — **не коммитить** (добавить
    в `.gitignore` или держать вне репозитория).
- **Проверки зелёные:** `go build ./...`, `go test ./...`, `go vet ./...`;
  `make build-pi` собирает `bin/edge-pi` (~16 МБ).
- **`-race` пока НЕ проверен:** на Windows требует gcc/cgo (нет). Планируется
  через Docker (`docker run --rm -v ${PWD}:/app -w /app golang:1.26 go test -race ./...`)
  или на Pi/CI. Docker Desktop требует включить WSL2 (`wsl --install` от админа).
- **Следующий шаг (на выбор):**
  1. **Запуск на реальном Pi** — сборка схемы со светодиодами (резисторы 220 Ом,
     маркировка `220R`), `make build-pi`, systemd. **Ближайшая цель.**
  2. **Safety** — `usecase/safety` + `delivery/watchdog` (Safe Mode по
     устареванию данных / ошибке датчика). В плане это п.5.
  3. **Settings HTTP** — `usecase/settings` + `delivery/http` + middleware
     «только LAN» (смена порогов по сети).
- **Окружение:** терминал — PowerShell (не bash). Команды писать под PowerShell.
- **Важно:** `hardware: gpio` на ноутбуке даёт понятную ошибку (заглушка
  `gpio_other.go`) — это ожидаемо; для разработки на ноутбуке `hardware: console`.

## Цель MVP (текущий этап)

Устройство читает показания датчиков **из CSV-файла** (имитация), применяет
правила и **зажигает светодиоды** (имитация оборудования). Данные копятся в
SQLite и досылаются на Backend по HTTP с дедупликацией.

## Технологический стек (Edge)

- **Go** (на ноутбуке go1.27.3 windows/amd64; на Pi — Raspberry Pi OS).
- **Модуль:** `github.com/Anton-Babaev/AquaMind`
- **Репозиторий:** https://github.com/Anton-Babaev/AquaMind.git
- **SQLite:** `modernc.org/sqlite` (чистый Go, без CGO — важно для ARM).
- **GPIO:** `go-rpio` (чистый Go, без CGO) — для Этапа 4.
- **Архитектура:** Clean Architecture, слои `delivery → usecase → adapter`,
  порты объявляются в `usecase/<сценарий>/ports.go`, реализации — в `adapter/`.
- **Общий код между сервисами:** только `internal/contract/syncv1`.

## Схема подключения светодиодов (Этап 4)

| Светодиод | Смысл | GPIO (BCM) | Физический пин |
|---|---|---|---|
| 🟡 Жёлтый | Подогрев | GPIO17 | пин 11 |
| 🔴 Красный | Тревога | GPIO27 | пин 13 |
| 🟢 Зелёный | Норма | GPIO22 | пин 15 |
| ⚫ GND | Земля | — | пин 9 |

Схема на каждый светодиод (резистор обязателен, 220–330 Ом):
```
GPIO-пин ──[ резистор 220–330 Ом ]──▶|── GND
                              (анод) (катод)
```
Длинная ножка = анод (+), короткая = катод (−). Pi 3B+ GPIO = 3.3 В, ≤16 мА на пин.

## Порядок правил (приоритеты, §5.6)

1. Safe Mode (данные устарели / ошибка датчика) → подогрев выкл, алерт
2. Нет потока при включённом насосе → подогрев выкл, алерт
3. Критически высокая t° → подогрев выкл, алерт
4. Критически низкая t° → подогрев вкл, алерт
5. Мало кислорода → алерт (на подогрев не влияет)
6. Низкая t° → подогрев вкл ниже порога, выкл выше «порог + гистерезис»

## Пороги (configs/edge.yaml, черновые — заменить на реальные позже)

```
temp_low: 22.0, temp_low_hysteresis: 1.0
temp_critical_low: 18.0, temp_critical_high: 30.0
o2_low: 5.0, o2_low_hysteresis: 0.5
flow_min: 5.0
```

## Статус этапов

### ✅ Этап 1 — Домен + политики + тесты (готово)
- `internal/edge/domain/`: `reading.go`, `actuator.go`, `alert.go`, `thresholds.go`
- `internal/edge/domain/policy/`: `heating.go` (HeatingPolicy), `alarm.go` (AlarmPolicy)
- Тесты: `heating_test.go` (9 кейсов), `alarm_test.go` (8 кейсов) — зелёные
- `internal/edge/adapter/hardware/simulation/provider.go` — CSV-провайдер
  (ускорение времени, зацикливание, время = now()), 7 тестов

### ✅ Этап 2 — Control + Poller + console-LED (готово)
- `internal/edge/usecase/control/ports.go` — порты SensorSource, Actuators, Store, AlertTracker
- `internal/edge/usecase/control/control.go` — ControlUsecase (тик без блокировки при чтении)
- `internal/edge/usecase/alert/tracker.go` — AlertTracker (raised/cleared, антиспам), 4 теста
- `internal/edge/adapter/hardware/console/leds.go` — «лампочки» в терминале
- `internal/edge/delivery/poller/poller.go` — SensorPoller
- `internal/edge/adapter/storage/memory/store.go` — in-memory Store (временный)
- `internal/edge/config/config.go` — конфиг (env-override)
- `cmd/edge/main.go` — DI + graceful shutdown
- Проверено вживую: no_flow.csv → 🟢→🔴→🟢; day_cycle.csv → 🟡 при падении t°

### ✅ Этап 3 — SQLite + Sync + Backend-заглушка (готово)
- `internal/contract/syncv1/sync.go` — SyncBatch, SyncAck, TickDTO и др.
- `internal/edge/adapter/storage/sqlite/` — repository.go (WAL, миграции, SaveTick
  одной транзакцией), migrations/001_init.sql, thresholds.go
- `cmd/backend/main.go` — заглушка POST /api/v1/sync с дедупом по client_id+at
- `internal/edge/adapter/backendclient/client.go` — HTTP-клиент
- `internal/edge/delivery/syncworker/worker.go` — батчи из outbox, MarkSent
- `cmd/edge/main.go` — обновлён: SQLite + SyncWorker в горутине
- Проверено вживую: батчи принимаются; дедуп: повтор → accepted=0, duplicates=1

### ✅ Этап 4 — GPIO на Pi + настройка железа (готово)
- `internal/edge/adapter/hardware/gpio/` — адаптер на `go-rpio` (BCM 17/27/22),
  реализует порт `Actuators` (тот же интерфейс, что console):
  - `gpio.go` — общая (тестируемая) логика: `Pins`, `LEDs.Apply`, `LEDs.Close`,
    интерфейс `pinBackend`; `DefaultPins` = 17/27/22.
  - `gpio_linux.go` (build tag `linux`) — реальный backend на `go-rpio`
    (`rpio.Open`, `Output`, `High/Low`, `Close`).
  - `gpio_other.go` (build tag `!linux`) — заглушка для ноутбука: `New` возвращает
    понятную ошибку «only available on linux/arm», чтобы `go build ./...` и
    `go test ./...` проходили на Windows/macOS без железа.
  - `gpio_test.go` — 8 тестов на fake-backend (маппинг состояния → пины,
    context, ошибки записи, Close гасит всё и идемпотентен).
- Выбор адаптера по конфигу: `hardware: console|gpio` в `configs/edge.yaml`
  (+ секция `gpio:` с пинами). Env-override: `AQUAMIND_HARDWARE`.
  `config.Config` получил `Hardware HardwareKind` и `GPIOPins`; `validate()`
  проверяет значение. `cmd/edge/main.go` — `buildActuators()` выбирает адаптер,
  для gpio возвращает `Close()` (гасит светодиоды на SIGTERM).
- Инструкция по настройке Pi: `docs/RASPBERRY_PI_SETUP.md` (прошивка флешки,
  первый запуск, SSH, установка Go, копирование кода, `make build-pi`, права
  на GPIO, подключение светодиодов, systemd, диагностика, чек-лист).
- Кросс-компиляция `make build-pi` (GOOS=linux GOARCH=arm GOARM=7) — проверена,
  `bin/edge-pi` ~16 МБ. `go-rpio` и `modernc.org/sqlite` — чистый Go, без CGO.
- Проверено: `go build ./...`, `go test ./...`, `go vet ./...` — зелёные;
  на ноутбуке `hardware: gpio` даёт понятную ошибку, `hardware: console` работает.

### ✅ Этап 4.1 — Тесты адаптеров + Makefile-цели (готово, не закоммичено)
- `internal/edge/adapter/backendclient/client_test.go` — 6 тестов на `httptest`:
  успех (проверка метода/пути/Content-Type/тела/ack), 500, битый JSON ack,
  отмена ctx, таймаут, отказ соединения.
- `internal/edge/adapter/storage/sqlite/repository_test.go` — 8 тестов на
  `t.TempDir()`: применение миграций, SaveTick пишет reading+outbox (проверка
  payload), MarkSent убирает из pending, пустые ids, лимит PendingOutbox,
  round-trip порогов, upsert порогов, отмена ctx.
- `Makefile`: `test-race` (`go test -race ./...`), `cover` (coverage.out +
  coverage.html), `clean` чистит coverage-файлы.
- Все новые тесты зелёные (`go test ./...`), `go vet ./...` чисто.

### Опционально (если успеем)
- SafetyWatchdog (Safe Mode по устареванию данных) — `delivery/watchdog`
- Settings HTTP (смена порогов по LAN) — `delivery/http` + `usecase/settings`
- Минимальный frontend — `cmd/frontend`

## Как запускать (ноутбук, Windows/PowerShell)

```powershell
# Сборка
go build ./...
go test ./...

# Запуск Backend-заглушки (терминал 1)
go run ./cmd/backend

# Запуск Edge (терминал 2), сценарий через env
$env:AQUAMIND_SCENARIO="scenarios/no_flow.csv"
$env:AQUAMIND_POLL_INTERVAL="500ms"
$env:AQUAMIND_SYNC_INTERVAL="2s"
go run ./cmd/edge
```

Env-переменные: `AQUAMIND_SCENARIO`, `AQUAMIND_POLL_INTERVAL`,
`AQUAMIND_STORAGE`, `AQUAMIND_BACKEND_URL`, `AQUAMIND_SYNC_INTERVAL`,
`AQUAMIND_HARDWARE` (`console`|`gpio`).

## Поток загрузки на Pi

1. Разработка на ноутбуке → `git push` на GitHub.
2. На Pi (по SSH): `git clone` / `git pull`.
3. На Pi: `make build-pi` → бинарник `bin/edge-pi`.
4. systemd-сервис `deploy/edge/aquamind-edge.service` (SIGTERM → безопасное состояние).

## Структура каталогов (ключевое)

```
cmd/{edge,backend,frontend}/        # три программы
configs/{edge,backend}.yaml
scenarios/{day_cycle,no_flow}.csv
deploy/edge/aquamind-edge.service
internal/
  contract/syncv1/                  # общий контракт Edge↔Backend
  edge/
    config/
    domain/ + domain/policy/
    usecase/{control,alert,safety,settings,uplink}
    delivery/{poller,watchdog,syncworker,http}
    adapter/
      hardware/{simulation,gpio,console,modbus}
      storage/{sqlite,memory}
      backendclient/
  backend/                          # зона Романа (пока только заглушка в cmd/backend)
  frontend/
```

## Проверки перед коммитом

```powershell
go build ./...
go test ./...
go vet ./...
# опционально, если есть gcc/Docker/Pi:
# go test -race ./...
# make cover
```

## Заметки для нового чата

- **Ближайшая задача:** собрать схему на Pi (светодиоды + резисторы 220 Ом,
  маркировка `220R`; `100K`/`220K` — НЕ подходят), запустить `bin/edge-pi`.
- **Резистор 220 Ом:** полоски `красный-красный-коричневый-золотой`; проверить
  мультиметром (~220 Ом). Нужно 3 шт. (жёлтый/красный/зелёный).
- **Пины:** GPIO17=пин11 (жёлтый), GPIO27=пин13 (красный), GPIO22=пин15
  (зелёный), GND=пин9. См. `docs/RASPBERRY_PI_SETUP.md`.
- **`-race`:** локально на Windows не работает без gcc; вариант — Docker
  (`docker run --rm -v ${PWD}:/app -w /app golang:1.26 go test -race ./...`)
  или запуск на Pi. Docker Desktop требует WSL2 (`wsl --install` от админа).
- **Не коммитить** папку `Lessons/`.
