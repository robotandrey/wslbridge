# wslbridge

`wslbridge` — CLI-утилита для прокидывания **всего сетевого трафика из WSL / Ubuntu через SOCKS5-прокси**, работающий на хостовой системе (Windows / macOS), например под корпоративным VPN.

Основной сценарий:
- VPN запущен на хосте
- SOCKS-прокси доступен из WSL (часто через default gateway)
- WSL должен иметь доступ к внутренним сервисам (БД, API и т.д.)

---

## Как это работает

1. В WSL создаётся `tun`-интерфейс (`tun0`)
2. Default route переключается на `tun0`
3. `tun2socks` прокидывает трафик из `tun0` в SOCKS-прокси
4. Весь трафик WSL идёт через VPN хоста

---

## Требования

- Linux (Ubuntu / WSL2)
- `sudo` доступ
- SOCKS5-прокси, доступный из WSL
- Go ≥ 1.22 (для установки `tun2socks`)

---

## Установка

```bash
git clone https://github.com/<your-org>/wslbridge.git
cd wslbridge
make install
