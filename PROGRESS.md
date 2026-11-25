# AquaMind — карта проекта и статус

> **Быстрый старт для ИИ-агента:** этот файл — точка входа для продолжения работы.
> Прочитай его первым: здесь карта проекта, статус этапов, коммиты и грабли.
> Дипломный проект: автоматизация рыбной фермы (УЗВ) на Raspberry Pi 3B+.
> Работают два человека. **Наша зона — Edge-контроллер на Raspberry Pi (Go).**
> Backend (PostgreSQL, Telegram) — зона второго разработчика (Роман).

## Где мы сейчас (для продолжения в новом чате)

- **Готово:** Этапы 1–9 (домен+политики, Control+Poller+console, SQLite+Sync+
  Backend-заглушка, GPIO на Pi + README + инструкция по Pi, YAML-парсер,
  веб-интерфейс дашборд+пороги, kiosk на Pi, мелкие правки UI/документации,
  **полноценный Backend: PostgreSQL + Telegram**, **устойчивость Edge:
  CIDR-фильтр, retention, ретраи** — см. Этап 9).
- **ВАЖНО — структура каталогов изменилась:** код теперь лежит в корне модуля
  (`edge/...`, `backend/...`, `contract/...`), а НЕ в `internal/edge/...` и
  `internal/backend/...`. Старые пути в разделах Этапов 1–8 ниже — исторические;
  актуальная структура — в разделе «Структура каталогов (ключевое)».
- **Реальные датчики и исполнители пропускаем:** физически есть только
  светодиоды-индикаторы (подогрев/тревога/норма). Реальные исполнители (насос,
  подсветка, кормушка) — через модуль реле, когда придёт. Modbus-адаптер не делаем.
- **Проверки зелёные:** `go build ./...`, `go vet ./...`, `go test ./...`.
- **Добавлено после Этапа 4 (закоммичено в `cfb1363`):**
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
- **ГОТОВО И ПРОВЕРЕНО ВЖИВУЮ НА РЕАЛЬНОМ Pi (24.11.2026):**
  - Pi 3B+ прошит (Raspberry Pi OS Lite 64-bit, aarch64), SSH включён.
  - Сеть: раздача Wi-Fi с Android-телефона (2.4 ГГц!), Pi получает IP по DHCP
    (в сессии был `10.44.8.214`; IP может меняться — искать по MAC
    `b8-27-eb-13-76-83` через `arp -a`).
  - Код склонирован в `/opt/aquamind` (git clone).
  - `bin/edge-pi` собран на ноутбуке (`go build` с GOOS=linux GOARCH=arm GOARM=7)
    и перенесён через `scp`.
  - **Светодиоды РАБОТАЮТ на реальном железе:** зелёный горит всегда, жёлтый
    мигает по сценарию `day_cycle.csv`. Схема и пины подтверждены.
  - **systemd-сервис `aquamind-edge` работает и автономен:** `enabled`,
    `active (running)`, переживает `reboot`. Запускается с
    `Environment=AQUAMIND_HARDWARE=gpio` (добавлено в unit-файл).
  - **ВАЖНО (баг):** `configs/edge.yaml` **НЕ читается** — `config.Load()`
    берёт `Default()` и переопределяет только через env `AQUAMIND_*`.
    Флаг `--config` в unit-файле игнорируется. Поэтому `hardware: gpio` в файле
    не действует — задаётся через `Environment=AQUAMIND_HARDWARE=gpio`.
    **Нужен YAML-парсер** (см. план ниже).
  - **Права:** `data/edge.db` должен принадлежать `aquamind` (после ручного
    `sudo ./edge-pi` файл создаётся от root → `readonly database`). Лечится
    `sudo chown -R aquamind:aquamind /opt/aquamind`.
  - Backend недоступен (`localhost:8080` на Pi) — ожидаемо, данные копятся в
    SQLite (outbox). Это корректное автономное поведение.
- **Следующий шаг (см. «План работ» ниже):**
  1. ✅ **YAML-парсер конфига** — ГОТОВО (см. Этап 5 ниже).
  2. ✅ **Веб-интерфейс Edge (дашборд + пороги)** — ГОТОВО и закоммичено
     (`fc33c6d`, см. Этап 6 ниже). API `/api/state`, `/api/thresholds` проверены вживую.
     ✅ **Доработки UI и порогов** (Этап 6.1, см. ниже) — ГОТОВО и закоммичено.
  3. ✅ **Kiosk на Pi** — ГОТОВО (см. Этап 7 ниже). Pi на Raspberry Pi OS
     Desktop, chromium kiosk автозапускается на `http://localhost:8081`,
     дашборд открывается сам при загрузке на весь экран.
  4. **Актуаторы** (насос, подсветка, кормушка) — реальные исполнители через реле.
     **Ждём модуль реле** (ещё не куплен/не пришёл).
  5. **Кормление**, **Wi-Fi**, **Логи** (вкладки-заглушки уже есть), **Backend**, **Frontend**.
- **Окружение:** терминал — PowerShell (не bash). Команды писать под PowerShell.
- **Важно:** `hardware: gpio` на ноутбуке даёт понятную ошибку (заглушка
  `gpio_other.go`) — это ожидаемо; для разработки на ноутбуке `hardware: console`.

## Новая архитектура (согласована 24.11.2026)

Трёхуровневая система (соответствует Clean Architecture проекта):

1. **Edge (Raspberry Pi 3B+)** — работает **автономно**:
   - читает датчики, управляет исполнительными устройствами (сейчас светодиоды,
     позже — насос, подсветка, кормушка);
   - **локальный интерфейс на мониторе** (GUI с вкладками, клавиатура):
     дашборд, настройки порогов, подключение к Wi-Fi, режимы кормления, логи;
   - копит данные в SQLite и отправляет на Backend (outbox, досылка).
2. **Backend** (зона Романа) — приём данных, обработка, алерты (Telegram),
   хранение (PostgreSQL). Роман подключается удалённо.
3. **Frontend** — запросы к Backend API, таблицы/графики для удалённого
   пользователя.

### Решение по интерфейсу Edge (монитор)

- **Выбран Вариант A — веб-интерфейс в браузере (kiosk-режим).**
  - Pi поднимает локальный HTTP-сервер (Go, лёгкий, без CGO).
  - На мониторе — браузер в полноэкранном kiosk-режиме, выглядит как
    приложение с вкладками (Дашборд / Пороги / Wi-Fi / Кормление / Логи).
  - Плюсы: лёгкий для Pi 3B+, современный вид, работает и локально, и удалённо,
    не ломает кросс-компиляцию.
  - **Нюанс:** для kiosk нужен браузер (Desktop-образ или установка chromium
    отдельно). Сейчас стоит Lite — перейти на Desktop/поставить браузер
    на этапе реализации UI.
- Отклонены: TUI (просили «не консоль»), нативный GUI Fyne/Gio (тяжело для
  Pi 3B+, CGO, сложная сборка).

### План работ (приоритет)

1. **YAML-парсер конфига** — чтобы `configs/edge.yaml` реально читался
   (сейчас игнорируется). Критично для UI и настроек.
2. **Веб-интерфейс Edge** — `delivery/http` + статика (HTML/CSS/JS), вкладки:
   дашборд (показания, состояние светодиодов, алерты), пороги, Wi-Fi, кормление,
   логи. Kiosk-браузер на мониторе.
3. **Актуаторы (реальные исполнители)** — расширить порт `Actuators`
   (`SetPump`, `SetLight`, `Feed`, `SetAlarm`) поверх текущих светодиодов;
   управление через **модуль реле с оптронной развязкой** (насос, аэратор,
   подсветка) + PWM для кормушки-сервопривода. См. раздел «Реальные исполнители».
4. **Настройки** — `usecase/settings` + сохранение в SQLite (пороги уже есть),
   API для смены порогов из UI.
5. **Кормление** — расписание кормления (`usecase/feeding`).
6. **Backend** — Роман поднимает, указываем его адрес в конфиге (`backend.url`).
7. **Frontend** — удалённый пользователь (браузер → Backend API).

## Реальные исполнители (цель — дешёвые аналоги серьёзных механизмов)

**Решение (согласовано):** реальные датчики — НЕ обязательны (достаточно CSV-
имитации). Но **исполнители делаем настоящие** — на дешёвом аквариумном
оборудовании как аналогах промышленных механизмов УЗВ.

**Что есть физически (аквариум пользователя):**
- насос (циркуляция) — аналог насоса УЗВ;
- аэратор (компрессор) — аналог системы оксигенации;
- подсветка — аналог освещения;
- кормушка — аналог автокормушки.

**Как управлять с Pi (дёшево и безопасно):**
- **Модуль реле с оптронной развязкой** — коммутирует нагрузку 220 В по сигналу
  GPIO (тот же `High/Low`, что у светодиодов). Оптрон защищает Pi от 220 В.
- **Что заказать:** «4-канальный модуль реле 5V с оптронной развязкой,
  AC 250 В / 10 А на канал» (если устройств 5+ — брать 8-канальный).
  - Питание модуля — **5 В** с пина Pi (пин 2 или 4), GND — общий (пин 6),
    входы IN1–IN4 — на GPIO.
  - На каждый канал 3 контакта: `COM` / `NO` / `NC`. Для «вкл при сигнале» —
    `COM`+`NO` (разрываем один провод устройства 220 В).
  - **⚠️ Логика входов (active-low):** в описании модуля «IN срабатывает при
    замыкании на корпус» → реле включается при **низком уровне (GND/0 В)**.
    Значит в адаптере реле сигнал, вероятно, надо **инвертировать** относительно
    светодиодов (там «вкл» = High). Заложить в конфиг флаг `active_low: true`
    и проверить экспериментально на реальном модуле (подать High/Low на IN1).
  - **Питание Pi:** катушки 4 реле при срабатывании дают просадку по 5 В.
    Нужен блок питания Pi ≥2.5 А (или отдельное питание 5 В для модуля с общим
    GND). Иначе при одновременном включении нескольких реле Pi может
    перезагрузиться.
- **Кормушка:** если это **сервопривод** (поворот) — реле НЕ подойдёт, нужен
  PWM-драйвер (отдельный тип выхода). Если **моторчик** (крутится N секунд) —
  реле подходит. Уточнить модель кормушки.
- **Безопасность 220 В:** модуль с оптронной развязкой защищает Pi, но НЕ
  человека. 220 В часть — в закрытом корпусе, без оголённых проводов. По
  возможности начинать с 12 В устройств (аквариумные насосы часто 12 В).

**Влияние на архитектуру:**
- Порт `Actuators` расширяется до реальных механизмов:
  `SetPump(on)`, `SetLight(on)`, `Feed()`, `SetAlarm(on)` (вместо/в дополнение
  к текущему `Apply(state)` для светодиодов).
- Адаптеры: `console` (печать в терминал), `gpio` (пины реле + сервопривод).
- Пины реле задаются в `configs/edge.yaml` (секция `actuators:` — добавить при
  реализации). YAML-парсер (пункт 1) — фундамент и для этого.
- Светодиоды (подогрев/тревога/норма) остаются как **индикаторы** рядом с реле.

## Цель MVP (текущий этап)

Устройство читает показания датчиков **из CSV-файла** (имитация), применяет
правила и **зажигает светодиоды** (имитация оборудования). Данные копятся в
SQLite и досылаются на Backend по HTTP с дедупликацией.

## Технологический стек (Edge)

- **Go** (на ноутбуке go1.27.3 windows/amd64; на Pi — Raspberry Pi OS).
  В `go.mod` директива `go 1.26.0` (минимальная версия языка для модуля).
- **Модуль:** `github.com/Anton-Babaev/AquaMind`
- **Репозиторий:** https://github.com/Anton-Babaev/AquaMind.git
- **SQLite:** `modernc.org/sqlite` (чистый Go, без CGO — важно для ARM).
- **GPIO:** `go-rpio` (чистый Go, без CGO) — для Этапа 4.
- **YAML:** `gopkg.in/yaml.v3` (чистый Go) — для парсера конфига (Этап 5).
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
6. Много кислорода → алерт (warning)
7. pH ниже нормы → алерт (warning)
8. pH выше нормы → алерт (warning)
9. Низкая t° → подогрев вкл ниже порога, выкл выше «порог + гистерезис»

## Пороги (configs/edge.yaml, черновые — заменить на реальные позже)

```
# Температура, °C
temp_low: 22.0, temp_low_hysteresis: 1.0
temp_critical_low: 18.0, temp_critical_high: 30.0
# Кислород, мг/л
o2_low: 5.0, o2_high: 9.0, o2_low_hysteresis: 0.5
# Поток, л/мин
flow_min: 5.0
# pH (допустимый диапазон)
ph_low: 6.5, ph_high: 8.5
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

### ✅ Этап 4.1 — Тесты адаптеров + Makefile-цели (готово, закоммичено `cfb1363`)
- `internal/edge/adapter/backendclient/client_test.go` — 6 тестов на `httptest`:
  успех (проверка метода/пути/Content-Type/тела/ack), 500, битый JSON ack,
  отмена ctx, таймаут, отказ соединения.
- `internal/edge/adapter/storage/sqlite/repository_test.go` — 8 тестов на
  `t.TempDir()`: применение миграций, SaveTick пишет reading+outbox (проверка
  payload), MarkSent убирает из pending, пустые ids, лимит PendingOutbox,
  round-trip порогов, upsert порогов, отмена ctx.
- `Makefile`: `test-race` (`go test -race ./...`), `cover` (coverage.out +
  coverage.html), `clean` удаляет каталог `bin/` (coverage-файлы `clean`
  **не** трогает — их удаляют вручную при необходимости).
- Все новые тесты зелёные (`go test ./...`), `go vet ./...` чисто.

### ✅ Этап 5 — YAML-парсер конфига (готово, закоммичено `cac9d62`)
- `internal/edge/config/config.go` — переписан: теперь `configs/edge.yaml`
  **реально читается** (раньше игнорировался).
  - Зависимость `gopkg.in/yaml.v3` (чистый Go, без CGO — важно для ARM).
  - `Load(path string)` — три источника по приоритету: **дефолты → YAML-файл →
    env `AQUAMIND_*`** (env важнее файла). `Load("")` = дефолты + env (как раньше).
  - `LoadFromArgs(args)` — разбор флага `--config <путь>` (для systemd-юнита).
  - «Сырые» структуры `fileConfig`/`fileBackend`/... с тегами `yaml:"..."` и
    полями-указателями (nil = «в файле не задано» → остаётся дефолт).
  - Обёртка `duration` (реализует `yaml.Unmarshaler`) — парсит `"10s"`, `"500ms"`.
  - Читаются все секции файла: `gpio`, `backend`, `polling`, `safety`,
    `scenario`, `storage`, `control_plane`, `thresholds`. Секции `safety`,
    `storage.busy_timeout/retention`, `control_plane` пока только читаются в
    структуру (подключаются к логике на будущих этапах).
- `cmd/edge/main.go` — `config.Load()` → `config.LoadFromArgs(os.Args[1:])`.
- `internal/edge/config/config_test.go` — 10 тестов: дефолты, чтение файла,
  частичный файл (дефолты для отсутствующих полей), env важнее файла, чтение
  реального `configs/edge.yaml`, ошибки (битый YAML, неверный hardware,
  неверная длительность, невалидные пороги), отсутствующий файл, разбор
  `--config`, запуск без флага.
- Проверено: `go build ./...`, `go vet ./...`, `go test ./...` — зелёные.
- **Эффект:** теперь `hardware: gpio` в `configs/edge.yaml` действует без
  `Environment=AQUAMIND_HARDWARE=gpio`. Env-переменные остаются как override.

### ✅ Этап 6 — Веб-интерфейс Edge: дашборд + пороги (готово, закоммичено `fc33c6d`)
- **Слой доставки** `internal/edge/delivery/http/`:
  - `server.go` — `http.Server` на `cfg.ControlPlane.Listen` (по умолчанию
    `:8081`), graceful shutdown по ctx, `ReadHeaderTimeout`. Статика встроена
    через `//go:embed static` (один бинарник, ничего не копировать на Pi).
  - `handlers.go` — JSON API:
    - `GET /api/state` — показания + состояние выходов + активные алерты.
    - `GET /api/alerts?limit=N` — история событий алертов (raised/cleared),
      свежие первыми (источник — `AlertHistoryProvider`, реализуется SQLite).
    - `GET /api/thresholds` — текущие пороги.
    - `POST /api/thresholds` — сохранить пороги (валидация `Thresholds.Validate`).
    - DTO отделены от домена (`stateDTO`, `readingDTO`, `actuatorsDTO`, `alertDTO`).
  - `static/index.html` — дашборд с вкладками: Дашборд / Пороги / Wi-Fi /
    Кормление / Логи (последние три — заглушки).
  - `static/style.css` — тёмный дашборд (крупные цифры, LED-индикаторы).
  - `static/app.js` — переключение вкладок, опрос `/api/state` каждые 2 с,
    индикатор соединения, форма порогов (load/save).
- **`ControlUsecase`** (`usecase/control/control.go`): добавлены поля
  `lastReading`, `lastAlerts`, `hasReading`; в `Tick` они обновляются под
  блокировкой; новые методы `LastReading() (SensorReading, bool)` и
  `LastAlerts() []Alert` (копия) — для UI.
- **`domain.Thresholds`**: добавлены JSON-теги (`temp_low`, `flow_min`, ...),
  чтобы API отдавал snake_case.
- **`cmd/edge/main.go`**: `edgehttp.NewServer(cfg.ControlPlane.Listen, ctrl, store, store)`
  (4 аргумента: `listen`, `state`=ControlUsecase, `thresholds`=хранилище,
  `history`=хранилище) запускается в отдельной горутине рядом с SyncWorker и Poller.
- **Проверено вживую** (ноутбук, `hardware: console`): `GET /api/state` вернул
  корректный JSON (temp 24.1, flow 12, o2 7.2, ph 7.2, normal=true).
  `go build ./...`, `go vet ./...`, `go test ./...` — зелёные.
- **Как открыть:** запустить `go run ./edge/cmd/edge`, в браузере
  `http://localhost:8081`.
- **Осталось по UI:** вкладки Wi-Fi/Кормление/Логи (заглушки), графики истории,
  kiosk-настройка на Pi (Desktop + chromium).

### ✅ Этап 6.1 — Доработки UI и порогов (готово, закоммичено `fc33c6d`)
- **Пороги: новые поля O₂-верх и pH-диапазон.**
  - `domain.Thresholds`: добавлены `O2High`, `PHLow`, `PHHigh` (JSON-теги
    `o2_high`, `ph_low`, `ph_high`).
  - `Thresholds.Validate()` — человекочитаемые сообщения с числами и единицами
    (например, «нижний порог кислорода (5.0 мг/л) должен быть ниже верхнего
    (9.0 мг/л)», «границы pH должны быть в диапазоне от 0 до 14»).
  - `config.go` — дефолты (`o2_high: 9.0`, `ph_low: 6.5`, `ph_high: 8.5`),
    YAML-структура и `applyTo` для новых полей.
  - `configs/edge.yaml` — новые пороги с комментариями по группам.
  - `sqlite/thresholds.go` — маппинг новых ключей (`o2_high`, `ph_low`,
    `ph_high`) + дефолты.
- **Алерты: новые правила.**
  - `domain/policy/alarm.go`: `o2_high` (warning), `ph_low` (warning),
    `ph_high` (warning). Обновлён doc-комментарий приоритетов.
  - `handlers.go`: убран технический префикс `invalid thresholds:` из ответа
    API — пользователь видит только понятный текст валидации.
- **Дашборд: индикация «мало/норма/высоко» для всех датчиков.**
  - Под каждым показанием — цветная подпись-статус:
    - Температура: «критически низкая» / «мало» / «норма» / «критически высокая»
      (по `temp_critical_low`, `temp_low`, `temp_critical_high`).
    - Поток: «мало» / «норма» (по `flow_min`).
    - Кислород: «мало» / «норма» / «высоко» (по `o2_low`, `o2_high`).
    - pH: «низкий» / «норма» / «высокий» (по `ph_low`, `ph_high`).
  - Пороги подгружаются с `/api/thresholds` и кэшируются на клиенте
    (`refreshThresholdsCache` раз в 5 с + обновление после сохранения).
- **Форма порогов: группировка.** Поля разбиты на `fieldset` по группам:
  Температура / Кислород / Поток / pH (заголовки + рамки).
- **Убрана нижняя панель индикаторов** (кружки Подогрев/Тревога/Норма/Safe
  Mode) — HTML, JS (`setLed`/`setStateText`) и CSS (`.indicators`/`.led`/
  `.ind-*`) удалены. Состояние выходов остаётся доступно в `/api/state`.
- **Исправлена ложная ошибка «Не удалось загрузить пороги»** при открытии
  вкладки: фоновая подгрузка больше не пишет ошибку в форму; сообщение
  сбрасывается при успешной загрузке, реальная ошибка видна только при
  сохранении.
- **Тесты:** обновлены `testThresholds` (новые поля) и `alarm_test.go`
  (добавлен `PH` в показания + кейсы `o2_high`, `ph_low`, `ph_high`).
- **Проверено:** `go build ./...`, `go vet ./...`, `go test ./...` — зелёные;
  UI проверен вживую (edge перезапущен, статика встроена в бинарник).

### ✅ Этап 7 — Kiosk на Raspberry Pi (готово, проверено вживую 25.11.2026)
- **Pi перепрошит на Raspberry Pi OS Desktop (64-bit)** — вместо Lite (нужен
  браузер для kiosk). При прошивке через Raspberry Pi Imager заданы: пользователь
  `aquamind`, SSH (password auth), Wi-Fi домашней сети, hostname `aquamind`.
- **Сеть:** домашний Wi-Fi `192.168.1.x`, Pi получил IP `192.168.1.101` (DHCP,
  может меняться — искать по MAC `b8-27-eb-13-76-83` через `arp -a` или смотреть
  IP на экране Pi при загрузке).
- **Восстановление после перепрошивки** (флешка стёрта → всё пропало):
  - `sudo apt update && sudo apt install -y git`
  - `sudo git clone https://github.com/Anton-Babaev/AquaMind.git /opt/aquamind`
  - `sudo chown -R aquamind:aquamind /opt/aquamind`
  - бинарник собран на ноутбуке и залит по `scp` (см. заметки ниже).
  - systemd-сервис восстановлен: `sudo cp deploy/edge/aquamind-edge.service
    /etc/systemd/system/` + `daemon-reload` + `enable` + `start`.
- **Kiosk-автозапуск** (создан удалённо по SSH, без ручного набора на Pi):
  - `~/.config/autostart/aquamind-kiosk.desktop`:
    ```
    [Desktop Entry]
    Type=Application
    Name=AquaMind Kiosk
    Exec=chromium --kiosk --noerrdialogs --disable-infobars --incognito --password-store=basic http://localhost:8081
    X-GNOME-Autostart-enabled=true
    ```
  - `--password-store=basic` — убирает всплывающее окно GNOME Keyring при
    автозагрузке (браузер не трогает keyring).
  - **Проверено:** после `sudo reboot` дашборд открывается сам на весь экран.
  - Выход из kiosk: `Alt+F4`.
- **Ярлык на рабочем столе** `~/Desktop/aquamind.desktop` (двойной клик →
  дашборд в kiosk). Создаётся через `cat > ... <<'EOF'` (удобнее, чем `nano`).
- **⚠️ ВАЖНЫЕ ГРАБЛИ (сборка/деплой на Pi):**
  1. **Сборка под ARM в PowerShell** — только `$env:GOOS="linux"`,
     `$env:GOARCH="arm"`, `$env:GOARM="7"`. Команда `set GOOS=...` (cmd-синтаксис)
     в PowerShell **НЕ работает** → собирается PE32+ под Windows → на Pi
     `systemctl` падает с `status=203/EXEC`. Проверка: `go version -m bin\edge-pi`
     должна показать `GOOS=linux GOARCH=arm GOARM=7`; на Pi `file edge-pi` →
     `ELF 32-bit LSB executable, ARM`.
  2. **Бинарник кладём в `/opt/aquamind/edge-pi`** (в корень, куда смотрит
     `ExecStart` в unit-файле), а НЕ в `bin/`. Иначе `203/EXEC`.
  3. **Права на запись:** `/opt/aquamind` после `sudo git clone` принадлежит
     root → `scp` даёт `Permission denied`. Лечится
     `sudo chown -R aquamind:aquamind /opt/aquamind`.
  4. **`~` без слэша** (`~Desktop`) не раскрывается → `No such file or directory`.
     Писать `~/Desktop/...`.
- **Проверено вживую:** `systemctl status aquamind-edge` → `active (running)`,
  светодиоды мигают (`норма:🟢`), `curl localhost:8081/api/state` отдаёт JSON,
  дашборд открывается в kiosk сам после reboot.

### ✅ Этап 7.1 — Мелкие правки UI и документации (готово)
- **История алертов на дашборде:** убраны непонятные надписи «снят»/«поднят»
  рядом со временем события. В `static/app.js` (`makeAlertItem`) для истории
  теперь выводится только время (`fmtTime(al.at)`), без текста состояния.
  Активные алерты по-прежнему помечаются «активен». Данные истории берутся из
  `GET /api/alerts?limit=N` (см. Этап 6).
- **`docs/RASPBERRY_PI_SETUP.md` переписан под актуальный процесс:**
  - Прошивка **флешки (microSD)** через Raspberry Pi Imager — образ
    **Raspberry Pi OS Desktop (64-bit)** вместо Lite (нужен браузер для kiosk).
  - Go на Pi больше не ставим — сборка на ноутбуке (кросс-компиляция).
  - Сборка в **PowerShell** через `$env:GOOS/GOARCH/GOARM` + грабля про
    `set` vs `$env:` и `203/EXEC`.
  - Бинарник в `/opt/aquamind/edge-pi` (в корне), проверка `file` → `ELF ... ARM`.
  - Новый раздел **§12 Kiosk**: автозапуск chromium, ярлык на рабочем столе,
    `--password-store=basic`, `~/Desktop` со слэшем.
  - Диагностика дополнена (`203/EXEC`, `Permission denied` на scp, `~Desktop`,
    keyring, kiosk), чек-лист обновлён под Desktop + kiosk.
- **Применение на Pi:** статика встроена в бинарник (`//go:embed`) → пересобрать
  `edge-pi`, залить `scp`, `sudo systemctl restart aquamind-edge`.

### ✅ Этап 8 — Backend: PostgreSQL + Telegram (готово, зона Романа)
Полноценный Backend вместо заглушки `cmd/backend`. Clean Architecture,
слои `delivery → usecase → adapter`, порты в `usecase/*/ports.go`.

- **Конфиг** `internal/backend/config/config.go` — по образцу Edge:
  дефолты → YAML (`configs/backend.yaml`) → env `AQUAMIND_BACKEND_*`.
  Секции: `listen`, `postgres` (dsn/max_conns/connect_timeout), `telegram`
  (token/chat_id/timeout/enabled), `notify` (cooldown/notify_cleared).
  Секреты (DSN, токен) на хостинге — только через env. 10 тестов.
- **Домен** `internal/backend/domain/domain.go` — `Reading`, `ActuatorState`,
  `Alert` (типы/уровни/состояния как в Edge), `Tick`.
- **PostgreSQL** `internal/backend/adapter/postgres/` (драйвер `pgx/v5`):
  - `migrations/001_init.sql` — таблицы `clients`, `ticks`, `alerts`;
    уникальный индекс `(client_id, at_ms)` на `ticks` для дедупликации.
  - `repository.go` — `Open` (пул + ping + миграции из `//go:embed`),
    `SaveBatch` (одна транзакция: upsert клиента, вставка тиков с
    `ON CONFLICT DO NOTHING ... RETURNING id`, алерты только для новых тиков;
    возвращает accepted/duplicates), `RecentTicks`, `Ping`.
  - `errors.go` — `isNoRows` (pgx.ErrNoRows → дубликат).
- **Usecase ingest** `internal/backend/usecase/ingest/` — `Handle(SyncBatch)`:
  DTO→домен, `SaveBatch`, уведомления только по алертам из принятых тиков,
  возврат `SyncAck`. Порты `Store`, `Notifier`. 4 теста (fake store/notifier).
- **Usecase notify** `internal/backend/usecase/notify/` — антиспам:
  raised отправляется один раз в пределах `cooldown` на `(client, type)`;
  cleared сбрасывает состояние и (если включено) шлёт «восстановлено».
  Состояние в памяти (MVP). 5 тестов.
- **Адаптер Telegram** `internal/backend/adapter/telegram/sender.go` —
  `sendMessage` через `net/http`, `baseURL` подменяется в тестах. 3 теста
  на `httptest` (успех, ok=false, HTTP 401).
- **HTTP delivery** `internal/backend/delivery/http/server.go` —
  `POST /api/v1/sync` (приём пакета, 500 при ошибке → Edge повторит, дедуп
  защитит), `GET /healthz` (ping БД).
- **`cmd/backend/main.go`** — DI: postgres → notifier (Telegram или лог-заглушка)
  → ingest → HTTP-сервер; graceful shutdown по SIGTERM.
- **Деплой:** `deploy/backend/Dockerfile` (multi-stage, CGO off, alpine),
  `deploy/backend/env.example` (пример env, `.env` в `.gitignore`).
- **Проверено:** `go build ./...`, `go vet ./...`, `go test ./...` — зелёные
  (все тесты backend проходят без реальной БД/Telegram).
- **Осталось (по желанию):** API для Frontend (чтение истории/графиков),
  SMS-канал, персистентное состояние антиспама в БД.

### ✅ Этап 9 — Устойчивость Edge: CIDR, retention, ретраи (готово)
Закрыты «висящие» поля конфига и добавлена устойчивость к сбоям сети.

- **CIDR-фильтр для веб-интерфейса Edge** (`edge/delivery/http/middleware.go`):
  - `cidrFilter` — middleware, проверяющий `RemoteAddr` по подсетям из
    `ControlPlane.AllowedCIDRs`. Пустой список → доступ всем. Некорректные
    CIDR пропускаются с логом.
  - Смотрит на `RemoteAddr`, а НЕ на `X-Forwarded-For` (заголовок подделывается
    клиентом).
  - `NewServer` принимает `allowedCIDRs` и оборачивает mux фильтром.
  - `middleware_test.go` — 6 кейсов + тест на игнор `X-Forwarded-For`.
- **Retention (очистка устаревших данных)** — закрывает неиспользуемый
  `Storage.Retention`:
  - `sqlite.Repository.Purge(ctx, retention)` — удаляет старые `readings`/
    `alerts` и **отправленные** `outbox` (по `sent_at_ms`). Неотправленные
    outbox не трогает (at-least-once).
  - `edge/delivery/retention/retention.go` — `Worker`, периодический триггер
    очистки (раз в час или чаще при малом retention).
  - `purge_test.go` — удаление старого/сохранение свежего, noop при retention=0,
    сохранение неотправленных outbox.
- **Ретраи в `backendclient`** (устойчивость к сбоям сети):
  - `Sync` повторяет запрос при сетевых ошибках и 5xx (экспоненциальный backoff,
    3 попытки по умолчанию). 4xx и битый JSON — без повтора. Прерывается по
    отмене контекста.
  - `retry_test.go` — успех после 5xx, отсутствие ретрая на 4xx, исчерпание
    попыток, остановка по ctx.
- **`cmd/edge/main.go`** — подключены `retention` (в горутине) и `AllowedCIDRs`
  в http-сервер.
- **Тесты Backend** (закрыты пробелы покрытия):
  - `backend/usecase/query/query_test.go` — фейковый Reader: Clients/State/
    Ticks/Alerts, нормализация limit, ошибки, пустой client_id.
  - `backend/delivery/http/server_test.go` — httptest: sync (ok/405/400/500),
    healthz (ok/503), clients, state (400/404/ok), ticks/alerts (400),
    parseMillis/parseInt.
- **Проверено:** `go build ./...`, `go vet ./...`, `go test ./...` — зелёные.

### Опционально (если успеем)
- SafetyWatchdog (Safe Mode по устареванию данных) — `delivery/watchdog`
- Settings HTTP (смена порогов по LAN) — `delivery/http` + `usecase/settings`

## Как запускать (ноутбук, Windows/PowerShell)

```powershell
# Сборка
go build ./...
go test ./...


# Запуск Edge (терминал 2), сценарий через env
$env:AQUAMIND_SCENARIO="edge/scenarios/no_flow.csv"
$env:AQUAMIND_POLL_INTERVAL="500ms"
go run ./edge/cmd/edge
```

`AQUAMIND_STORAGE`, `AQUAMIND_BACKEND_URL`, `AQUAMIND_SYNC_INTERVAL`,
`AQUAMIND_HARDWARE` (`console`|`gpio`).

## Поток загрузки на Pi

1. Разработка на ноутбуке → `git push` на GitHub.
2. На Pi (по SSH): `git clone` / `git pull`.
3. На Pi: `make build-pi` → бинарник `bin/edge-pi`.

4. systemd-сервис `edge/deploy/aquamind-edge.service` (SIGTERM → безопасное состояние).

## Структура каталогов (ключевое)

> **Актуально:** код в корне модуля (`edge/`, `backend/`, `contract/`), а не в
> `internal/`. Пути в описаниях Этапов 1–8 выше (`internal/edge/...`) — исторические.

```

contract/syncv1/                    # общий контракт Edge↔Backend
edge/                               # контур Edge (Raspberry Pi)
  cmd/edge/                         #   точка входа Edge
  config/ + configs/edge.yaml       #   конфиг (дефолты→YAML→env)
  scenarios/{day_cycle,no_flow}.csv #   CSV-сценарии симуляции
  deploy/aquamind-edge.service      #   systemd-сервис
  domain/ + domain/policy/
  usecase/{control,alert,safety,settings,uplink}
    # реализованы: control, alert, safety, settings, uplink
  delivery/{poller,watchdog,syncworker,retention,http}
    # реализованы: poller, watchdog, syncworker, retention, http
    # http/: server.go, handlers.go, middleware.go (CIDR), middleware_test.go,
    #        static/ (дашборд, //go:embed); middleware/ — пустая папка-заглушка (.gitkeep)
  adapter/
    hardware/{simulation,gpio,console,modbus}
      # реализованы: simulation, gpio, console; modbus/ — пустая заглушка (.gitkeep), не делаем
    storage/{sqlite,memory}
    backendclient/                  # HTTP-клиент с ретраями (Этап 9)
backend/                            # контур Backend (сервер) + встроенный Frontend
  cmd/backend/                      #   точка входа Backend
  config/ + configs/backend.yaml    #   конфиг (дефолты→YAML→env)
  deploy/{Dockerfile,docker-compose.yml,env.example}
  domain/                           #   Reading, ActuatorState, Alert, Tick
  usecase/{ingest,notify,query,alert}
    # реализованы: ingest, notify, query; alert/ — пустая заглушка (.gitkeep), зарезервирован
  adapter/{postgres,telegram}/      #   pgx/v5 + миграции; Telegram sendMessage
  delivery/http/                    #   server.go, dto.go, server_test.go
    static/                         #   Frontend: статика дашборда (//go:embed)
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

- **Ближайшая задача:** **Актуаторы** (реле с оптронной развязкой:
  насос/аэратор/подсветка + кормушка). **Ждём модуль реле** — как придёт,
  начинаем. Пока можно готовить код с флагом `active_low` и секцией `actuators:`
  в конфиге. См. «План работ» и «Реальные исполнители».
- **Реальные датчики/Modbus — пропускаем.** Физически есть только светодиоды-
  индикаторы. CSV-симуляция остаётся источником показаний.
- **Этап 9 закрыл:** CIDR-фильтр (`AllowedCIDRs`), retention (`Storage.Retention`),
  ретраи `backendclient`, тесты `query` и `delivery/http` Backend.
- **PROJECT_SNAPSHOT.md — устаревший черновик** (пути `internal/...`, выводы
  «AllowedCIDRs/Retention не применяются» уже неверны). Источник истины — код;
  при необходимости пересоздать снимок заново.
- **Kiosk на Pi готов** (Этап 7): дашборд автозапускается на весь экран.
- **Светодиоды на Pi уже собраны и работают** (см. блок «ГОТОВО И ПРОВЕРЕНО
  ВЖИВУЮ НА РЕАЛЬНОМ Pi»).
- **IP Pi:** `192.168.1.101` (домашний Wi-Fi, DHCP). SSH из cmd/PowerShell:
  `ssh aquamind@192.168.1.101`.
- **Сборка/деплой на Pi** — см. грабли в Этапе 7 (PowerShell `$env:GOOS`,
  бинарник в `/opt/aquamind/edge-pi`, `chown`, `~/Desktop`).
- **Резистор 220 Ом:** полоски `красный-красный-коричневый-золотой`; проверить
  мультиметром (~220 Ом). Нужно 3 шт. (жёлтый/красный/зелёный).
- **Пины:** GPIO17=пин11 (жёлтый), GPIO27=пин13 (красный), GPIO22=пин15
  (зелёный), GND=пин9. См. `docs/RASPBERRY_PI_SETUP.md`.
- **`-race`:** локально на Windows не работает без gcc; вариант — Docker
  (`docker run --rm -v ${PWD}:/app -w /app golang:1.26 go test -race ./...`)
  или запуск на Pi. Docker Desktop требует WSL2 (`wsl --install` от админа).
- **Не коммитить** папку `Lessons/`.
