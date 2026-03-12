#wslbridge

`wslbridge` — CLI-утилита, которая прокидывает **весь сетевой трафик из WSL/Ubuntu через SOCKS5-прокси**, запущенный на хостовой системе (обычно Windows под корпоративным VPN).

Поддержка сейчас сфокусирована на WSL2/Ubuntu. macOS/Windows-флоу будет добавлен отдельно.

---

## Как это работает

1. В WSL создаётся `tun`-интерфейс (`tun0`)
2. Default route переключается на `tun0`
3. `tun2socks` прокидывает трафик из `tun0` в SOCKS-прокси
4. Весь трафик WSL идёт через VPN хоста

---

## Требования

- Linux (Ubuntu / WSL2)
- Доступ `sudo`
- SOCKS5-прокси, доступный из WSL (часто через default gateway)
- Go ≥ 1.22 (нужен для установки `tun2socks` при первом запуске)

---

## Установка

```bash
git clone https://github.com/<your-org>/wslbridge.git
cd wslbridge
make install   # бинарь попадет в ~/.local/bin/wslbridge
```

Убедитесь, что `~/.local/bin` есть в `$PATH`.

---

## Быстрый старт (WSL)

1) Запустите Shadowsocks/другой SOCKS5 на хосте. Обычно он слушает на `172.xx.xx.1:1080` (адрес default gateway в WSL).

2) Выполните:

```bash
wslbridge init --force --socks-port=1080
```

Команда:
- попросит указать DNS для WSL (попадает в `/etc/resolv.conf`)
- определит IP SOCKS-шлюза (по текущему default route или `eth0`),
- создаст конфиг `.values/values.local.yaml` в корне проекта,
- настроит `tun0` и запустит `tun2socks` с логами в `/tmp/tun2socks.log`.

Повторный запуск без `--force` пропустит перезапуск, если `tun2socks` уже работает и default route указывает на `tun0`.

---

## Команды

- `wslbridge init` — настройка DNS/роутинга и запуск `tun2socks`
- `wslbridge status` — показать текущий статус
- `wslbridge stop` — остановить `tun2socks` и вернуть маршруты
- `wslbridge help` — краткая справка по доступным командам

---

## Полезные флаги

- `--skip-deps` — не устанавливать системные зависимости через `apt`
- `--force` — форсировать вопросы/перезапуск `tun2socks`
- `--socks-port=<n>` — указать порт SOCKS-прокси без интерактивного ввода

---

## Где что лежит

- Конфиг: `<repo>/.values/values.local.yaml`
- Состояние: `~/.local/state/wslbridge/` (`default_route.txt`, `tun2socks.pid`)
- Логи `tun2socks`: `/tmp/tun2socks.log`

---

## Диагностика

- Посмотреть статус: `wslbridge status`
- Остановить и вернуть маршруты: `wslbridge stop`
- Проверить маршрут по умолчанию: `ip route show default`
- Проверить, что `tun2socks` жив: `kill -0 $(cat ~/.local/state/wslbridge/tun2socks.pid)`
- Если трафик не идёт, загляните в `/tmp/tun2socks.log` и убедитесь, что SOCKS-прокси доступен из WSL.

Пример:

```bash
wslbridge status
Config: /home/<user>/source/repos/go/wslbridge/.values/values.local.yaml
WSL: true
Default route: default dev tun0 scope link
Default is tun: true
Tun dev: tun0
Tun link: yes
SOCKS: 172.30.112.1:1080
Tun2socks running: yes
Tun2socks pid file: /home/<user>/.local/state/wslbridge/tun2socks.pid
Tun2socks pid: 12345
```
