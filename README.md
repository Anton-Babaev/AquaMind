# AquaMind

**Открытый аппаратно-программный комплекс для автоматизации аквакультуры** —
установок замкнутого водоснабжения (УЗВ) и аквариумов. Написан на Go.

> Дипломный проект буткемпа VK Education, трек «Микросервисы на Go».
> Команда: Антон Бабаев, Роман Забелин.

---

## О проекте

Рынок УЗВ растёт, а автоматизация остаётся «узким горлышком»: коммерческие
решения закрыты и привязывают к поставщику (vendor lock-in), а open-source
проекты «игрушечные» и монолитные. **AquaMind** — попытка сделать
профессиональный open-source комплекс на Go: надёжность промышленного уровня
при доступности свободного ПО.

### Ключевая идея

**Edge-контроллер на Raspberry Pi работает автономно.** Он читает датчики,
применяет правила безопасности, управляет оборудованием и копит данные локально.
Связь с сервером — только для синхронизации истории и алертов. Если сервер
недоступен, ферма продолжает работать: правила и светодиоды не зависят от сети.

---

## Архитектура (кратко)

Два контура, один Go-модуль, две программы в `cmd/`:

```
                       ┌────────────── Raspberry Pi 3 B+ ──────────────┐
 Ноутбук в LAN ──HTTP──▶│  Edge (Go)  ◀── CSV-сценарий                  │
 (настройка порогов,    │     │  ▲                                      │
  только LAN)           │     ▼  │                                      │
                        │  GPIO-светодиоды      SQLite (WAL)           │
                        └─────┬─────────────────────────────────────────┘
                              │ HTTP POST · JSON (push) · SyncBatch
                              ▼
                        Backend (Go) ──▶ PostgreSQL
                              │
                              ├──▶ Telegram API ──▶ фермер
                              │
                              └──▶ Frontend (встроен в Backend)
                                     статика + API, браузер оператора
```

| Сервис | Где работает | Роль | Хранит данные |
|---|---|---|---|
| **Edge** | Raspberry Pi 3 B+ | Автономный источник правды: читает датчики, применяет правила, управляет выходами, копит и досылает данные | SQLite |
| **Backend** | Сервер / ноутбук | Принимает пакеты с дедупликацией, хранит историю, шлёт алерты в Telegram, отдаёт веб-дашборд оператора | PostgreSQL |

**Frontend — часть Backend-контура.** Отдельного сервиса нет: статика
(HTML/CSS/JS) встроена в бинарник Backend через `//go:embed` и раздаётся тем же
HTTP-сервером, что и API. Backend и Frontend разворачиваются вместе на сервере.

**Направление связей:** Edge → Backend. Edge не открывает порты в интернет,
Backend не ходит к Edge.

### Слои Edge (Clean Architecture)

```
delivery → usecase → adapter
```

- **Entity** (`domain/`, `domain/policy/`) — сущности и чистые функции-политики,
  только stdlib.
- **UseCase** (`usecase/<сценарий>/`) — сценарии: Control, Alert, Safety,
  Settings, Sync. Интерфейсы (порты) объявляет потребитель — `ports.go`.
- **Adapter** (`adapter/`) — реализации портов: CSV-симулятор, GPIO, SQLite,
  HTTP-клиент Backend.
- **Delivery** (`delivery/`) — триггеры: поллер, watchdog, sync-worker, HTTP.

**HAL — два интерфейса:** `SensorSource` (чтение) и `Actuators` (выходы).
Замена симулятора на реальные датчики (Modbus) не требует правки правил.

### Правила (порядок приоритетов)

| # | Условие | Подогрев | Индикация | Алерт |
|---|---|---|---|---|
| 1 | Safe Mode (данные устарели / ошибка датчика) | выкл | 🔴 | да |
| 2 | Нет потока при включённом насосе | выкл | 🔴 | да |
| 3 | Критически высокая t° | выкл | 🔴 | да |
| 4 | Критически низкая t° | вкл | 🔴 | да |
| 5 | Мало кислорода | не влияет | 🔴 | да |
| 6 | Низкая t° | вкл ниже порога, выкл выше «порог + гистерезис» | 🟡 | нет |

🟡 — подогрев включён, 🔴 — активный алерт, 🟢 — всё в норме.

---

## Быстрый старт (ноутбук)

Требуется **Go 1.26+**. Железо не нужно — по умолчанию используется
CSV-симуляция и «лампочки» в терминале.

```bash
# Сборка и тесты
make build
make test

# Терминал 1 — Backend-заглушка
make run-backend

# Терминал 2 — Edge (сценарий через env)
AQUAMIND_SCENARIO=edge/scenarios/no_flow.csv \
AQUAMIND_POLL_INTERVAL=500ms \
AQUAMIND_SYNC_INTERVAL=2s \
make run-edge
```

В терминале Edge появятся «светодиоды»:

```
LED  подогрев:⚪  тревога:⚪  норма:🟢
LED  подогрев:⚪  тревога:🔴  норма:⚪
```

### Переменные окружения

| Переменная | Назначение | По умолчанию |
|---|---|---|
| `AQUAMIND_SCENARIO` | путь к CSV-сценарию | `scenarios/day_cycle.csv` |
| `AQUAMIND_POLL_INTERVAL` | период опроса датчиков | `2s` |
| `AQUAMIND_STORAGE` | путь к SQLite | `data/edge.db` |
| `AQUAMIND_BACKEND_URL` | адрес Backend | `http://localhost:8080` |
| `AQUAMIND_SYNC_INTERVAL` | период синхронизации | `10s` |
| `AQUAMIND_HARDWARE` | адаптер выходов: `console` \| `gpio` | `console` |

> **Пути к сценариям.** Дефолт в коде — `scenarios/day_cycle.csv` (относительно
> рабочей директории). При запуске из корня репозитория задайте путь явно:
> `AQUAMIND_SCENARIO=edge/scenarios/day_cycle.csv`. На Pi сервис работает с
> `WorkingDirectory=/opt/aquamind`, где `scenarios/` лежит в корне (см.
> `docs/RASPBERRY_PI_SETUP.md`).

---

## Запуск на Raspberry Pi

1. Соберите бинарник под ARMv7:

   ```bash
   make build-pi        # → bin/edge-pi
   ```

2. Скопируйте на Pi и переключите `hardware: gpio` в `edge/configs/edge.yaml`.
3. Подключите светодиоды (BCM 17/27/22 через резисторы 220–330 Ом).
4. Установите systemd-сервис `edge/deploy/aquamind-edge.service`.

**Полная пошаговая инструкция:** [`docs/RASPBERRY_PI_SETUP.md`](docs/RASPBERRY_PI_SETUP.md)
(прошивка флешки, SSH, установка Go, права на GPIO, systemd, диагностика).

### Схема подключения светодиодов

| Светодиод | Смысл | GPIO (BCM) | Физ. пин |
|---|---|---|---|
| 🟡 Жёлтый | Подогрев | GPIO17 | пин 11 |
| 🔴 Красный | Тревога | GPIO27 | пин 13 |
| 🟢 Зелёный | Норма | GPIO22 | пин 15 |
| ⚫ GND | Земля | — | пин 9 |

```
GPIO-пин ──[ резистор 220–330 Ом ]──▶|── GND
                              (анод) (катод)
```

---

## Структура репозитория

```
aquamind/
├── contract/syncv1/   # единственный общий код Edge↔Backend
├── edge/              # Контур Edge (Raspberry Pi)
│   ├── cmd/edge/      #   точка входа Edge
│   ├── config/ + configs/edge.yaml
│   ├── scenarios/     #   CSV-сценарии симуляции
│   ├── deploy/        #   systemd-сервис Edge
│   ├── domain/ + domain/policy/
│   ├── usecase/{control,alert,safety,settings,uplink}
│   ├── delivery/{poller,watchdog,syncworker,retention,http}
│   │   └── http/      #   server.go, handlers.go, middleware.go (CIDR) + static/ (дашборд)
│   └── adapter/{hardware,storage,backendclient}
│       └── hardware/{simulation,gpio,console}  # modbus/ — пустая заглушка, не делаем
└── backend/           # Контур Backend (сервер) + встроенный Frontend
    ├── cmd/backend/   #   точка входа Backend
    ├── config/ + configs/backend.yaml
    ├── deploy/        #   Dockerfile + docker-compose + env.example
    ├── domain/
    ├── usecase/{ingest,notify,query}  # alert/ — пустая заглушка, зарезервирован
    ├── delivery/http/ #   server.go, dto.go + раздача статики (//go:embed static)
    │   └── static/    #   Frontend: index.html, app.js, style.css
    └── adapter/{postgres,telegram}
```

---

## Технологии

- **Go** — язык реализации всех сервисов.
- **SQLite** (`modernc.org/sqlite`) — чистый Go, без CGO (важно для ARM).
- **GPIO** (`go-rpio`) — чистый Go, без CGO.
- **PostgreSQL** — хранилище Backend.

- **HTML/CSS/JS** — Frontend, встроен в бинарник Backend (`//go:embed`).

---

## Документация

| Документ | Что внутри |
|---|---|
| [`docs/PROGRESS.md`](docs/PROGRESS.md) | Карта проекта и статус этапов — **читать первым при продолжении работы** |
| [`docs/RASPBERRY_PI_SETUP.md`](docs/RASPBERRY_PI_SETUP.md) | Настройка Raspberry Pi от флешки до systemd и kiosk |

---

## Статус

| Этап | Статус |
|---|---|
| 1. Домен + политики + тесты | ✅ готово |
| 2. Control + Poller + console-LED | ✅ готово |
| 3. SQLite + Sync + Backend-заглушка | ✅ готово |
| 4. GPIO на Pi + настройка железа | ✅ готово |
| 5. YAML-парсер конфига | ✅ готово |
| 6. Веб-интерфейс Edge (дашборд + пороги) | ✅ готово |
| 7. Kiosk на Raspberry Pi | ✅ готово |
| 8. Backend: PostgreSQL + Telegram | ✅ готово |
| 9. Устойчивость Edge: CIDR, retention, ретраи | ✅ готово |
| Реальные исполнители (реле) | ⏳ ждём модуль реле |

> Реальные датчики не обязательны — используется CSV-симуляция. Сейчас есть
> светодиоды-индикаторы (подогрев/тревога/норма); реальные исполнители (насос,
> подсветка, кормушка) подключаются через модуль реле, когда он придёт.

---

## Проверки перед коммитом

```bash
go build ./...
go test ./...
go vet ./...
```

---

## Лицензия

Проект развивается как open-source. Модель — open-source + services
(консалтинг, интеграция, поддержка). Код остаётся открытым.
