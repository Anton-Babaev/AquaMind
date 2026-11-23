# Настройка Raspberry Pi для AquaMind Edge

Пошаговая инструкция: от чистой флешки до работающего сервиса с мигающими
светодиодами. Рассчитана на Raspberry Pi 3 B+ и Raspberry Pi OS (Bookworm,
64-bit или 32-bit — обе подходят, бинарник собираем под ARMv7).

> Всё, что в блоках `на ноутбуке` — выполняем на своём компьютере,
> `на Pi` — по SSH на малине.

---

## 0. Что понадобится

- Raspberry Pi 3 B+ + блок питания 5 В / 2.5 А.
- microSD 8–32 ГБ (Class 10 / A1).
- Ноутбук с кардридером.
- 3 светодиода (жёлтый, красный, зелёный), 3 резистора 220–330 Ом,
  макетная плата и провода «папа-мама» (или пайка).
- Wi-Fi или Ethernet (для Pi 3 B+ удобнее Ethernet — стабильнее).

---

## 1. Прошивка microSD (на ноутбуке)

1. Скачайте **Raspberry Pi Imager**: https://www.raspberrypi.com/software/
2. Вставьте microSD в кардридер.
3. В Imager:
   - **Choose OS** → `Raspberry Pi OS (other)` → `Raspberry Pi OS Lite (64-bit)`
     (Lite — без графики, нам нужен только терминал).
   - **Choose Storage** → ваша microSD.
4. Нажмите **шестерёнку (⚙) / «Edit settings»** и задайте:
   - **Hostname:** `aquamind`
   - **Enable SSH** → `Use password authentication` (или положите публичный ключ).
   - **Username / Password:** например `aquamind` / надёжный пароль.
   - **Wireless LAN:** SSID и пароль Wi-Fi (если без Ethernet).
   - **Locale:** часовой пояс, раскладка (не критично).
5. **Write** → дождитесь окончания → извлеките карту.

> Если Imager старой версии без настроек — после записи создайте на разделе
> `boot` (FAT) пустой файл `ssh` (включит SSH) и `wpa_supplicant.conf` с Wi-Fi.

---

## 2. Первый запуск и SSH (на ноутбуке)

1. Вставьте microSD в Pi, подключите питание, подождите 1–2 минуты.
2. Найдите IP малины:
   - в роутере (список DHCP-клиентов, ищите `aquamind`), или
   - `ping aquamind.local` (mDNS; на Windows нужен Bonjour, часто уже есть).
3. Подключитесь по SSH:

```bash
ssh aquamind@aquamind.local
# или
ssh aquamind@192.168.1.42
```

4. Обновите систему (на Pi):

```bash
sudo apt update && sudo apt upgrade -y
sudo reboot
```

---

## 3. Установка Go (на Pi)

Ставим Go из официального архива (в apt версия устаревшая).

```bash
# Проверьте актуальную версию на https://go.dev/dl/ ; здесь — пример 1.23.x
GO_VER=1.23.4
ARCH=$(uname -m)   # aarch64 -> arm64, armv7l -> armv6l (для 32-bit)
case "$ARCH" in
  aarch64) GOARCH=arm64 ;;
  armv7l)  GOARCH=armv6l ;;
  *) echo "unknown arch $ARCH"; exit 1 ;;
esac

cd /tmp
wget "https://go.dev/dl/go${GO_VER}.linux-${GOARCH}.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VER}.linux-${GOARCH}.tar.gz"

# Добавляем в PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile
go version
```

> **Важно:** на самой малине Go нужен только если вы хотите собирать прямо на
> ней. Мы собираем на ноутбуке (кросс-компиляция, см. §5) — тогда Go на Pi
> не обязателен. Но он удобен для отладки.

---

## 4. Копирование кода на Pi

### Вариант А — через git (рекомендуется)

```bash
# на Pi
sudo mkdir -p /opt/aquamind
sudo chown $USER:$USER /opt/aquamind
git clone https://github.com/Anton-Babaev/AquaMind.git /opt/aquamind
cd /opt/aquamind
```

Обновление в будущем:

```bash
cd /opt/aquamind && git pull
```

### Вариант Б — копирование бинарника с ноутбука (без git на Pi)

```bash
# на ноутбуке (PowerShell)
scp bin/edge-pi aquamind@aquamind.local:/tmp/edge-pi
scp -r configs scenarios aquamind@aquamind.local:/tmp/aquamind-assets/
```

Затем на Pi разложить в `/opt/aquamind` (см. §6).

---

## 5. Сборка под Pi (на ноутбуке)

Кросс-компиляция под ARMv7 (Raspberry Pi 3 B+):

```bash
# на ноутбуке, из корня репозитория
make build-pi
# эквивалент: GOOS=linux GOARCH=arm GOARM=7 go build -o bin/edge-pi ./cmd/edge
```

Получится `bin/edge-pi` (~16 МБ). Перенесите его на Pi:

```bash
# на ноутбуке (PowerShell)
scp bin/edge-pi aquamind@aquamind.local:/tmp/edge-pi
```

> `go-rpio` — чистый Go без CGO, поэтому кросс-компиляция работает «из коробки».
> `modernc.org/sqlite` — тоже чистый Go. Никаких `CGO_ENABLED=1` не нужно.

---

## 6. Раскладка на Pi

```bash
# на Pi
sudo mkdir -p /opt/aquamind/{configs,scenarios,data}
sudo chown -R $USER:$USER /opt/aquamind

# если копировали бинарник:
cp /tmp/edge-pi /opt/aquamind/edge-pi
chmod +x /opt/aquamind/edge-pi

# конфиги и сценарии (из репозитория или скопированные)
cp -r /opt/aquamind/configs/* /opt/aquamind/configs/ 2>/dev/null || true
cp -r /opt/aquamind/scenarios/* /opt/aquamind/scenarios/ 2>/dev/null || true
```

Итоговая структура:

```
/opt/aquamind/
├── edge-pi
├── configs/edge.yaml
├── scenarios/day_cycle.csv
├── scenarios/no_flow.csv
└── data/edge.db          # создастся автоматически
```

---

## 7. Подключение светодиодов

**Схема на каждый светодиод** (резистор обязателен!):

```
GPIO-пин ──[ резистор 220–330 Ом ]──▶|── GND
                              (анод) (катод)
```

Длинная ножка = анод (+), короткая = катод (−).

| Светодиод | Смысл   | GPIO (BCM) | Физ. пин | Резистор |
|-----------|---------|------------|----------|----------|
| 🟡 Жёлтый | Подогрев| GPIO17     | пин 11   | 220–330 Ом |
| 🔴 Красный| Тревога | GPIO27     | пин 13   | 220–330 Ом |
| 🟢 Зелёный| Норма   | GPIO22     | пин 15   | 220–330 Ом |
| ⚫ GND    | Земля   | —          | пин 9    | —        |

Pi 3 B+ GPIO = 3.3 В, ≤16 мА на пин. Резистор 220–330 Ом ограничивает ток
до безопасного уровня (~10 мА).

> **Проверка полярности:** анод (длинная ножка) — к резистору/пину GPIO,
> катод (короткая, с «плоской» стороной корпуса) — к GND.

---

## 8. Настройка конфига на Pi

Отредактируйте `/opt/aquamind/configs/edge.yaml`:

```yaml
client_id: edge-01

# На Pi — реальные светодиоды:
hardware: gpio

gpio:
  heating: 17   # 🟡 подогрев, пин 11
  alarm: 27     # 🔴 тревога,  пин 13
  normal: 22    # 🟢 норма,    пин 15

backend:
  url: "http://<IP-ноутбука-с-backend>:8080"   # адрес Backend-заглушки/Романа
  sync_interval: 10s
  request_timeout: 5s
  batch_size: 100

polling:
  interval: 2s

scenario:
  path: "scenarios/day_cycle.csv"
  speed: 60
  loop: true

storage:
  path: "data/edge.db"
```

> Пути в конфиге относительные — сервис запускается с `WorkingDirectory=/opt/aquamind`,
> поэтому `scenarios/...` и `data/...` разрешаются относительно `/opt/aquamind`.

---

## 9. Права на GPIO

`go-rpio` маппит `/dev/gpiomem`, для этого нужны права root или членство в
группе `gpio`. Проще всего — запускать сервис от root (см. systemd ниже).
Альтернатива — добавить пользователя в группу:

```bash
sudo usermod -aG gpio aquamind
# перелогиниться
```

> На Raspberry Pi OS `/dev/gpiomem` обычно доступен группе `gpio`.
> Если сервис запускается от `aquamind` и падает с `open gpio (need root...)` —
> либо добавьте в группу `gpio`, либо запускайте от root.

---

## 10. Ручной запуск (проверка)

Перед установкой сервиса проверьте вручную:

```bash
cd /opt/aquamind
sudo ./edge-pi
```

Ожидаемый вывод:

```
hardware: gpio (BCM heating=17 alarm=27 normal=22)
edge started: client_id=edge-01 hardware=gpio scenario=scenarios/day_cycle.csv ...
```

Светодиоды должны реагировать на сценарий (например, `no_flow.csv` →
зелёный → красный → зелёный). Остановка — `Ctrl+C` (SIGINT) → все светодиоды
гаснут.

> Если видите `gpio adapter is only available on linux/arm` — вы запустили
> бинарник, собранный под другую платформу. Пересоберите `make build-pi`.

---

## 11. systemd-сервис (автозапуск)

Создайте пользователя-сервиса (если ещё нет) и установите unit:

```bash
# на Pi
sudo useradd -r -s /usr/sbin/nologin aquamind 2>/dev/null || true
sudo chown -R aquamind:aquamind /opt/aquamind

sudo cp /opt/aquamind/deploy/edge/aquamind-edge.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now aquamind-edge
```

Проверка:

```bash
systemctl status aquamind-edge
journalctl -u aquamind-edge -f      # живой лог
```

> Unit запускает сервис от пользователя `aquamind`. Если GPIO недоступен
> (нет прав на `/dev/gpiomem`), добавьте `aquamind` в группу `gpio`
> (`sudo usermod -aG gpio aquamind && sudo systemctl restart aquamind-edge`)
> либо в unit замените `User=aquamind` на `User=root`.

Управление:

```bash
sudo systemctl restart aquamind-edge   # перезапуск
sudo systemctl stop aquamind-edge      # остановка (SIGTERM → светодиоды гаснут)
sudo systemctl disable aquamind-edge   # убрать из автозапуска
```

---

## 12. Обновление кода на Pi

```bash
# на ноутбуке
git push
make build-pi
scp bin/edge-pi aquamind@aquamind.local:/tmp/edge-pi

# на Pi
sudo systemctl stop aquamind-edge
sudo cp /tmp/edge-pi /opt/aquamind/edge-pi
sudo chmod +x /opt/aquamind/edge-pi
sudo systemctl start aquamind-edge
```

Или через git (если клонировали репозиторий и собираете на Pi):

```bash
cd /opt/aquamind && git pull && make build-pi
sudo systemctl restart aquamind-edge
```

---

## 13. Диагностика (частые проблемы)

| Симптом | Причина | Решение |
|---|---|---|
| `open gpio (need root / group gpio?)` | нет прав на `/dev/gpiomem` | запуск от root или `usermod -aG gpio` |
| `gpio adapter is only available on linux/arm` | бинарник не под ARM | `make build-pi`, копировать `bin/edge-pi` |
| Светодиод не горит | перепутана полярность / нет резистора / не тот пин | проверить анод/катод, BCM-номер, резистор |
| Горит всегда | пин занят другой функцией (UART/SPI) | взять свободные GPIO (17/27/22 свободны) |
| `permission denied` на `data/edge.db` | владелец каталога | `chown -R aquamind:aquamind /opt/aquamind` |
| Сервис падает в цикле | ошибка конфига | `journalctl -u aquamind-edge -n 50` |
| Backend недоступен | неверный `backend.url` | указать IP ноутбука с запущенным `cmd/backend` |

Проверка GPIO вручную (без AquaMind):

```bash
# мигнуть GPIO17 (BCM) — если есть python3-gpiozero
sudo apt install -y python3-gpiozero
python3 -c "from gpiozero import LED; from time import sleep; l=LED(17); l.on(); sleep(2); l.off()"
```

---

## 14. Чек-лист готовности

- [ ] Pi загружается, SSH работает, система обновлена.
- [ ] `bin/edge-pi` собран через `make build-pi` и скопирован на Pi.
- [ ] Светодиоды подключены через резисторы (17/27/22 → GND).
- [ ] В `configs/edge.yaml` стоит `hardware: gpio`.
- [ ] Ручной запуск `sudo ./edge-pi` зажигает светодиоды по сценарию.
- [ ] systemd-сервис `aquamind-edge` включён и в статусе `active (running)`.
- [ ] `journalctl -u aquamind-edge` показывает тики без ошибок.
- [ ] Backend получает батчи (если запущен на ноутбуке).
