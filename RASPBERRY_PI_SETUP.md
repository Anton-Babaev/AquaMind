# Настройка Raspberry Pi для AquaMind Edge

Пошаговая инструкция: от чистой флешки (microSD) до работающего сервиса с
мигающими светодиодами **и дашбордом на мониторе в kiosk-режиме**. Рассчитана
на Raspberry Pi 3 B+ и **Raspberry Pi OS Desktop (64-bit)**.

> Всё, что в блоках `на ноутбуке` — выполняем на своём компьютере,
> `на Pi` — по SSH на малине.
>
> **Терминал на ноутбуке:** PowerShell (не cmd). Команды сборки написаны под
> PowerShell (`$env:GOOS=...`). В cmd было бы `set GOOS=...` — **не путать**,
> иначе соберётся бинарник под Windows и на Pi упадёт с `203/EXEC`.

---

## 0. Что понадобится

- Raspberry Pi 3 B+ + блок питания 5 В / 2.5 А.
- microSD (флешка) 8–32 ГБ (Class 10 / A1).
- Ноутбук с кардридером.
- **HDMI-монитор, клавиатура, мышь** (для kiosk-дашборда).
- 3 светодиода (жёлтый, красный, зелёный), 3 резистора 220–330 Ом,
  макетная плата и провода «папа-мама» (или пайка).
- Wi-Fi или Ethernet (для Pi 3 B+ удобнее Ethernet — стабильнее).

---

## 1. Прошивка флешки (microSD) — на ноутбуке

1. Скачайте **Raspberry Pi Imager**: https://www.raspberrypi.com/software/
2. Вставьте microSD в кардридер.
3. В Imager:
   - **Choose OS** → `Raspberry Pi OS (64-bit)` — это **Desktop**-версия
     (с рабочим столом и браузером; нужна для kiosk).
     > Не берите `Lite` — там нет графики, kiosk не запустить.
   - **Choose Storage** → ваша microSD.
4. Нажмите **шестерёнку (⚙) / «Edit settings»** и задайте:
   - **Hostname:** `aquamind`
   - **Enable SSH** → `Use password authentication` (или положите публичный ключ).
   - **Username / Password:** например `aquamind` / надёжный пароль.
     **Запомните пароль** — он же для `sudo` и SSH.
   - **Wireless LAN:** SSID и пароль Wi-Fi (домашняя сеть, **2.4 ГГц**).
   - **Wireless LAN country:** `RU` (или ваша страна).
   - **Locale:** часовой пояс `Europe/Moscow`, раскладка `us`.
5. **Write** → дождитесь окончания → извлеките флешку.

> Если Imager старой версии без настроек — после записи создайте на разделе
> `boot` (FAT) пустой файл `ssh` (включит SSH) и `wpa_supplicant.conf` с Wi-Fi.

---

## 2. Первый запуск и SSH (на ноутбуке)

1. Вставьте флешку в Pi, подключите **HDMI-монитор, клавиатуру, мышь**, затем
   питание.
2. Первая загрузка Desktop — 1–3 минуты. Должен появиться **рабочий стол**.
   IP малины обычно показан на экране при загрузке (или в правом верхнем углу).
3. Найдите IP малины:
   - на экране Pi при загрузке, или
   - в роутере (список DHCP-клиентов, ищите `aquamind`), или
   - `arp -a` на ноутбуке — ищите MAC `b8-27-eb-13-76-83`, или
   - `ping aquamind.local` (mDNS).
4. Подключитесь по SSH (PowerShell/cmd):

```powershell
ssh aquamind@192.168.1.101
# или
ssh aquamind@aquamind.local
```

> IP по DHCP может меняться. Если не подключается — перепроверьте IP на экране
> Pi или через `arp -a`.

5. Обновите систему (на Pi):

```bash
sudo apt update && sudo apt upgrade -y
sudo reboot
```

---

## 3. Go на Pi — НЕ нужен

Мы собираем бинарник **на ноутбуке** (кросс-компиляция под ARM, см. §5) и
заливаем по `scp`. Устанавливать Go на Pi не обязательно.

> Если всё же хотите собирать прямо на Pi — поставьте Go из архива
> (https://go.dev/dl/), но это не требуется для штатного процесса.

---

## 4. Копирование кода на Pi

### Вариант А — через git (рекомендуется)

```bash
# на Pi
sudo apt install -y git
sudo git clone https://github.com/Anton-Babaev/AquaMind.git /opt/aquamind
sudo chown -R aquamind:aquamind /opt/aquamind
```

> ⚠️ `chown` обязателен: после `sudo git clone` папка принадлежит root, и
> `scp` бинарника даст `Permission denied`.

Обновление в будущем:

```bash
cd /opt/aquamind && sudo git pull
```

### Вариант Б — только бинарник (без git на Pi)

См. §5–§6: собрать на ноутбуке и залить `scp` в `/opt/aquamind/edge-pi`.

---

## 5. Сборка под Pi (на ноутбуке, PowerShell)

Кросс-компиляция под ARMv7 (Raspberry Pi 3 B+). **В PowerShell** — через `$env:`:

```powershell
cd D:\VK\Golang\AquaMind\aquamind
$env:GOOS="linux"
$env:GOARCH="arm"
$env:GOARM="7"
go build -o bin\edge-pi ./edge/cmd/edge
```

> ⚠️ **Главная грабля:** в PowerShell команда `set GOOS=linux` (cmd-синтаксис)
> **НЕ работает** — переменная не выставится, и `go build` соберёт бинарник под
> Windows (`PE32+`). На Pi такой файл не запустится (`status=203/EXEC`).
> В PowerShell только `$env:GOOS="linux"`.

Проверьте, что собралось под ARM:

```powershell
go version -m bin\edge-pi
```

В выводе должно быть `GOOS=linux`, `GOARCH=arm`, `GOARM=7`.

> `go-rpio` и `modernc.org/sqlite` — чистый Go без CGO, кросс-компиляция
> работает «из коробки».

---

## 6. Раскладка на Pi

Бинарник кладём **в корень `/opt/aquamind/edge-pi`** — именно туда смотрит
`ExecStart` в unit-файле (НЕ в `bin/`, иначе `203/EXEC`).

```powershell
# на ноутбуке (PowerShell)
scp bin\edge-pi aquamind@192.168.1.101:/opt/aquamind/edge-pi
```

```bash
# на Pi
chmod +x /opt/aquamind/edge-pi
file /opt/aquamind/edge-pi
# ожидаем: ELF 32-bit LSB executable, ARM, EABI5
```

Итоговая структура:

```
/opt/aquamind/
├── edge-pi                    # бинарник (ARM)
├── edge/
│   ├── configs/edge.pi.yaml   # конфиг для Pi (gpio, пути, ::1/128)
│   ├── scenarios/day_cycle.csv
│   ├── scenarios/no_flow.csv
│   └── deploy/aquamind-edge.service
├── backend/                   # backend (если разворачиваем на Pi)
├── contract/                  # общий контракт syncv1
└── data/edge.db               # создастся автоматически
```

> Если `scp` даёт `Permission denied` — на Pi выполните
> `sudo chown -R aquamind:aquamind /opt/aquamind`.

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

## 8. Конфиг на Pi

Для Pi в репозитории есть **готовый конфиг** `edge/configs/edge.pi.yaml`
(в git, ничего править вручную не нужно). Он отличается от `edge.yaml`
(ноутбук) тремя вещами:

- `hardware: gpio` — реальные светодиоды;
- `scenario.path: "edge/scenarios/day_cycle.csv"` — путь под структуру репозитория;
- `allowed_cidrs` включает `::1/128` — kiosk-браузер на Pi ходит через IPv6.

systemd-юнит (`edge/deploy/aquamind-edge.service`) уже указывает на этот файл:

```
ExecStart=/opt/aquamind/edge-pi --config /opt/aquamind/edge/configs/edge.pi.yaml
WorkingDirectory=/opt/aquamind
```

> Пути в конфиге относительные — сервис запускается с `WorkingDirectory=/opt/aquamind`,
> поэтому `edge/scenarios/...` и `data/...` разрешаются относительно `/opt/aquamind`.
>
> Если нужно поменять адрес backend — правьте `backend.url` в
> `edge/configs/edge.pi.yaml` и коммитьте (или переопределяйте через env
> `AQUAMIND_*`, они важнее файла).

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
sudo ./edge-pi --config edge/configs/edge.pi.yaml
```

Ожидаемый вывод:

```
hardware: gpio (BCM heating=17 alarm=27 normal=22)
edge started: client_id=edge-01 hardware=gpio scenario=edge/scenarios/day_cycle.csv ...
```

Светодиоды должны реагировать на сценарий (например, `no_flow.csv` →
зелёный → красный → зелёный). Остановка — `Ctrl+C` (SIGINT) → все светодиоды
гаснут.

> Если видите `gpio adapter is only available on linux/arm` — вы запустили
> бинарник, собранный под другую платформу. Пересоберите (§5) и проверьте
> `file edge-pi` (должно быть `ELF ... ARM`).

---

## 11. systemd-сервис (автозапуск edge)

```bash
# на Pi
sudo cp /opt/aquamind/edge/deploy/aquamind-edge.service /etc/systemd/system/
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

## 12. Kiosk-дашборд на мониторе

Edge поднимает локальный веб-интерфейс на `http://localhost:8081`. На мониторе
Pi открываем его в **chromium в kiosk-режиме** (на весь экран, без рамок).

### 12.1. Проверка, что веб-интерфейс отвечает

```bash
curl -s http://localhost:8081/api/state
```

Должен вернуться JSON с показаниями. Если пусто — проверьте
`systemctl status aquamind-edge` и `journalctl -u aquamind-edge -n 50`.

### 12.2. chromium

На Desktop-образе chromium обычно уже есть. Проверьте:

```bash
which chromium || which chromium-browser
```

Если нет:

```bash
sudo apt update && sudo apt install -y chromium
```

### 12.3. Автозапуск kiosk (создаём удалённо по SSH)

Файл автозапуска создаётся одной командой (без ручного `nano`):

```bash
mkdir -p ~/.config/autostart
cat > ~/.config/autostart/aquamind-kiosk.desktop <<'EOF'
[Desktop Entry]
Type=Application
Name=AquaMind Kiosk
Exec=chromium --kiosk --noerrdialogs --disable-infobars --incognito --password-store=basic http://localhost:8081
X-GNOME-Autostart-enabled=true
EOF
```

> `--password-store=basic` — убирает всплывающее окно GNOME Keyring при
> автозагрузке (браузер не трогает keyring).

Проверка:

```bash
cat ~/.config/autostart/aquamind-kiosk.desktop
sudo reboot
```

После перезагрузки дашборд должен открыться **сам на весь экран**.
Выход из kiosk: `Alt+F4`.

> Автозапуск сработает, если Pi автоматически входит в графическую сессию
> (Desktop). Если при загрузке просит логин — сначала войдёт пользователь,
> потом откроется kiosk.

### 12.4. Ярлык на рабочем столе (опционально)

```bash
cat > ~/Desktop/aquamind.desktop <<'EOF'
[Desktop Entry]
Type=Application
Name=AquaMind
Comment=Дашборд AquaMind Edge
Exec=chromium --kiosk --noerrdialogs --disable-infobars --incognito --password-store=basic http://localhost:8081
Icon=utilities-system-monitor
Terminal=false
Categories=Utility;
EOF
chmod +x ~/Desktop/aquamind.desktop
gio set ~/Desktop/aquamind.desktop metadata::trusted true
```

> ⚠️ Пишите `~/Desktop/...` **со слэшем** после `~`. Без слэша (`~Desktop`)
> тильда не раскроется → `No such file or directory`.
> На русской локали папка может называться `Рабочий стол` — тогда используйте
> `~/"Рабочий стол"/aquamind.desktop`.

### 12.5. Отключить гашение экрана (опционально)

```bash
sudo raspi-config
```
→ `Display Options` → `Screen Blanking` → `No`.

---

## 13. Обновление кода на Pi

```powershell
# на ноутбуке (PowerShell)
cd D:\VK\Golang\AquaMind\aquamind
$env:GOOS="linux"; $env:GOARCH="arm"; $env:GOARM="7"; go build -o bin\edge-pi ./edge/cmd/edge
scp bin\edge-pi aquamind@192.168.1.101:/opt/aquamind/edge-pi
```

```bash
# на Pi (если менялся код — обновить и его)
cd /opt/aquamind && sudo git fetch --all --tags && sudo git reset --hard origin/master
sudo chown -R aquamind:aquamind /opt/aquamind
sudo systemctl restart aquamind-edge
```

> Конфиг Pi (`edge/configs/edge.pi.yaml`) и systemd-юнит теперь **в git** —
> после `reset --hard` ничего вручную править не нужно.

### Если `scp` в `/opt/aquamind/edge-pi` даёт `Permission denied`

Бывает, когда файл/каталог принадлежит root или `scp` не может перезаписать
занятый бинарник. Надёжный обход — залить во `/tmp` и переместить с `sudo`:

```powershell
# на ноутбуке (PowerShell)
scp bin\edge-pi aquamind@192.168.1.101:/tmp/edge-pi
```

```bash
# на Pi
sudo mv /tmp/edge-pi /opt/aquamind/edge-pi
sudo chmod +x /opt/aquamind/edge-pi
sudo systemctl restart aquamind-edge
```

> Статика веб-интерфейса **встроена в бинарник** (`//go:embed`), поэтому после
> изменения HTML/CSS/JS достаточно пересобрать и залить `edge-pi` — отдельно
> копировать `static/` не нужно.

---

## 14. Диагностика (частые проблемы)

| Симптом | Причина | Решение |
|---|---|---|
| `status=203/EXEC` в `systemctl status` | бинарник не под ARM (собран под Windows) или неверный путь | пересобрать в PowerShell через `$env:GOOS`; проверить `file edge-pi` → `ELF ... ARM`; бинарник в `/opt/aquamind/edge-pi` |
| `scp: dest open ... Permission denied` | `/opt/aquamind` принадлежит root | `sudo chown -R aquamind:aquamind /opt/aquamind` |
| `chmod: cannot access '~Desktop/...'` | нет слэша после `~` | писать `~/Desktop/...` |
| `open gpio (need root / group gpio?)` | нет прав на `/dev/gpiomem` | запуск от root или `usermod -aG gpio` |
| `gpio adapter is only available on linux/arm` | бинарник не под ARM | пересобрать (§5), проверить `file` |
| Светодиод не горит | перепутана полярность / нет резистора / не тот пин | проверить анод/катод, BCM-номер, резистор |
| Горит всегда | пин занят другой функцией (UART/SPI) | взять свободные GPIO (17/27/22 свободны) |
| `permission denied` на `data/edge.db` | владелец каталога | `chown -R aquamind:aquamind /opt/aquamind` |
| Сервис падает в цикле | ошибка конфига | `journalctl -u aquamind-edge -n 50` |
| Дашборд не открывается в kiosk | edge не запущен / не тот URL | `curl localhost:8081/api/state`; проверить `~/.config/autostart/aquamind-kiosk.desktop` |
| Окно GNOME Keyring при загрузке | chromium трогает keyring | флаг `--password-store=basic` в `Exec=` |
| Backend недоступен | неверный `backend.url` | указать IP ноутбука с запущенным `cmd/backend` |

Проверка GPIO вручную (без AquaMind):

```bash
# мигнуть GPIO17 (BCM) — если есть python3-gpiozero
sudo apt install -y python3-gpiozero
python3 -c "from gpiozero import LED; from time import sleep; l=LED(17); l.on(); sleep(2); l.off()"
```

---

## 15. Чек-лист готовности

- [ ] Флешка прошита **Desktop**-образом, Pi грузится в рабочий стол, SSH работает.
- [ ] Код в `/opt/aquamind`, владелец `aquamind` (`chown -R`).
- [ ] `bin/edge-pi` собран в PowerShell (`$env:GOOS=linux GOARCH=arm GOARM=7`),
      `file` на Pi → `ELF ... ARM`.
- [ ] Бинарник лежит в `/opt/aquamind/edge-pi` (в корне, не в `bin/`).
- [ ] Светодиоды подключены через резисторы (17/27/22 → GND).
- [ ] Сервис запускается с `--config edge/configs/edge.pi.yaml` (`hardware: gpio`).

- [ ] Ручной запуск `sudo ./edge-pi --config edge/configs/edge.pi.yaml` зажигает светодиоды по сценарию.
- [ ] systemd-сервис `aquamind-edge` включён и в статусе `active (running)`.
- [ ] `curl localhost:8081/api/state` отдаёт JSON.
- [ ] Kiosk-автозапуск настроен, после `reboot` дашборд открывается сам.
- [ ] Backend получает батчи (если запущен на ноутбуке).
