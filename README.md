#wslbridge

`wslbridge` вЂ” CLI-СѓС‚РёР»РёС‚Р°, РєРѕС‚РѕСЂР°СЏ РїСЂРѕРєРёРґС‹РІР°РµС‚ **РІРµСЃСЊ СЃРµС‚РµРІРѕР№ С‚СЂР°С„РёРє РёР· WSL/Ubuntu С‡РµСЂРµР· SOCKS5-РїСЂРѕРєСЃРё**, Р·Р°РїСѓС‰РµРЅРЅС‹Р№ РЅР° С…РѕСЃС‚РѕРІРѕР№ СЃРёСЃС‚РµРјРµ (РѕР±С‹С‡РЅРѕ Windows РїРѕРґ РєРѕСЂРїРѕСЂР°С‚РёРІРЅС‹Рј VPN).

РџРѕРґРґРµСЂР¶РєР° СЃРµР№С‡Р°СЃ СЃС„РѕРєСѓСЃРёСЂРѕРІР°РЅР° РЅР° WSL2/Ubuntu. macOS/Windows-С„Р»РѕСѓ Р±СѓРґРµС‚ РґРѕР±Р°РІР»РµРЅ РѕС‚РґРµР»СЊРЅРѕ.

---

## РљР°Рє СЌС‚Рѕ СЂР°Р±РѕС‚Р°РµС‚

1. Р’ WSL СЃРѕР·РґР°С‘С‚СЃСЏ `tun`-РёРЅС‚РµСЂС„РµР№СЃ (`tun0`)
2. Default route РїРµСЂРµРєР»СЋС‡Р°РµС‚СЃСЏ РЅР° `tun0`
3. `tun2socks` РїСЂРѕРєРёРґС‹РІР°РµС‚ С‚СЂР°С„РёРє РёР· `tun0` РІ SOCKS-РїСЂРѕРєСЃРё
4. Р’РµСЃСЊ С‚СЂР°С„РёРє WSL РёРґС‘С‚ С‡РµСЂРµР· VPN С…РѕСЃС‚Р°

---

## РўСЂРµР±РѕРІР°РЅРёСЏ

- Linux (Ubuntu / WSL2)
- Р”РѕСЃС‚СѓРї `sudo`
- SOCKS5-РїСЂРѕРєСЃРё, РґРѕСЃС‚СѓРїРЅС‹Р№ РёР· WSL (С‡Р°СЃС‚Рѕ С‡РµСЂРµР· default gateway)
- Go в‰Ґ 1.22 (РЅСѓР¶РµРЅ РґР»СЏ СѓСЃС‚Р°РЅРѕРІРєРё `tun2socks` РїСЂРё РїРµСЂРІРѕРј Р·Р°РїСѓСЃРєРµ)

---

## РЈСЃС‚Р°РЅРѕРІРєР°

```bash
git clone https://github.com/<your-org>/wslbridge.git
cd wslbridge
make install   # Р±РёРЅР°СЂСЊ РїРѕРїР°РґРµС‚ РІ ~/.local/bin/wslbridge
```

РЈР±РµРґРёС‚РµСЃСЊ, С‡С‚Рѕ `~/.local/bin` РµСЃС‚СЊ РІ `$PATH`.

---

## Р‘С‹СЃС‚СЂС‹Р№ СЃС‚Р°СЂС‚ (WSL)

1) Р—Р°РїСѓСЃС‚РёС‚Рµ Shadowsocks/РґСЂСѓРіРѕР№ SOCKS5 РЅР° С…РѕСЃС‚Рµ. РћР±С‹С‡РЅРѕ РѕРЅ СЃР»СѓС€Р°РµС‚ РЅР° `172.xx.xx.1:1080` (Р°РґСЂРµСЃ default gateway РІ WSL).

2) Р’С‹РїРѕР»РЅРёС‚Рµ:

```bash
wslbridge init --force --socks-port=1080
```

РљРѕРјР°РЅРґР°:
- РїРѕРїСЂРѕСЃРёС‚ СѓРєР°Р·Р°С‚СЊ DNS РґР»СЏ WSL (РїРѕРїР°РґР°РµС‚ РІ `/etc/resolv.conf`)
- РѕРїСЂРµРґРµР»РёС‚ IP SOCKS-С€Р»СЋР·Р° (РїРѕ С‚РµРєСѓС‰РµРјСѓ default route РёР»Рё `eth0`),
- СЃРѕР·РґР°СЃС‚ РєРѕРЅС„РёРі `.values/values.local.yaml` РІ РєРѕСЂРЅРµ РїСЂРѕРµРєС‚Р°,
- РЅР°СЃС‚СЂРѕРёС‚ `tun0` Рё Р·Р°РїСѓСЃС‚РёС‚ `tun2socks` СЃ Р»РѕРіР°РјРё РІ `/tmp/tun2socks.log`.

РџРѕРІС‚РѕСЂРЅС‹Р№ Р·Р°РїСѓСЃРє Р±РµР· `--force` РїСЂРѕРїСѓСЃС‚РёС‚ РїРµСЂРµР·Р°РїСѓСЃРє, РµСЃР»Рё `tun2socks` СѓР¶Рµ СЂР°Р±РѕС‚Р°РµС‚ Рё default route СѓРєР°Р·С‹РІР°РµС‚ РЅР° `tun0`.

---

## РљРѕРјР°РЅРґС‹

- `wslbridge init` вЂ” РЅР°СЃС‚СЂРѕР№РєР° DNS/СЂРѕСѓС‚РёРЅРіР° Рё Р·Р°РїСѓСЃРє `tun2socks`
- `wslbridge status` вЂ” РїРѕРєР°Р·Р°С‚СЊ С‚РµРєСѓС‰РёР№ СЃС‚Р°С‚СѓСЃ
- `wslbridge stop` вЂ” РѕСЃС‚Р°РЅРѕРІРёС‚СЊ `tun2socks` Рё РІРµСЂРЅСѓС‚СЊ РјР°СЂС€СЂСѓС‚С‹
- `wslbridge db init` — один раз настроить Warden URL/host и endpoint mask
- `wslbridge db start` — после перезагрузки/остановки заново поднять local DB proxy для всех добавленных баз
- `wslbridge db status` — показать состояние local DB proxy и список баз
- `wslbridge db stop` — остановить local DB proxy
- `wslbridge db add <service>` — добавить базу (service name), проверить endpoint TCP-коннект и сразу сделать доступной через local proxy
- `wslbridge db remove <service>` — удалить базу из списка
- `wslbridge help` вЂ” РєСЂР°С‚РєР°СЏ СЃРїСЂР°РІРєР° РїРѕ РґРѕСЃС‚СѓРїРЅС‹Рј РєРѕРјР°РЅРґР°Рј

---

## РџРѕР»РµР·РЅС‹Рµ С„Р»Р°РіРё

- `--skip-deps` вЂ” РЅРµ СѓСЃС‚Р°РЅР°РІР»РёРІР°С‚СЊ СЃРёСЃС‚РµРјРЅС‹Рµ Р·Р°РІРёСЃРёРјРѕСЃС‚Рё С‡РµСЂРµР· `apt`
- `--force` вЂ” С„РѕСЂСЃРёСЂРѕРІР°С‚СЊ РІРѕРїСЂРѕСЃС‹/РїРµСЂРµР·Р°РїСѓСЃРє `tun2socks`
- `--socks-port=<n>` вЂ” СѓРєР°Р·Р°С‚СЊ РїРѕСЂС‚ SOCKS-РїСЂРѕРєСЃРё Р±РµР· РёРЅС‚РµСЂР°РєС‚РёРІРЅРѕРіРѕ РІРІРѕРґР°

---

## Р“РґРµ С‡С‚Рѕ Р»РµР¶РёС‚

- РљРѕРЅС„РёРі: `<repo>/.values/values.local.yaml`
- Состояние: `~/.local/state/wslbridge/` (`default_route.txt`, `tun2socks.pid`, `db-proxy.pid`, `db-proxy.json`, `db-proxy.log`, `db-routes.json`)
- Р›РѕРіРё `tun2socks`: `/tmp/tun2socks.log`
- Логи proxy: `~/.local/state/wslbridge/db-proxy.log`

---

## Warden -> local proxy

Р”Р»СЏ РїРѕРґРєР»СЋС‡РµРЅРёСЏ Р±Р°Р· РёР· IDE Р±РµР· СЂСѓС‡РЅРѕРіРѕ `socat`:

```bash
wslbridge db init
wslbridge db add chatapi-ng
wslbridge db add bozon-saturn
```

`db init`:
- СЃРїСЂР°С€РёРІР°РµС‚ Warden URL/host;
- РµСЃР»Рё URL РїРѕР»РЅС‹Р№ (РІРєР»СЋС‡Р°СЏ `.../endpoints?service=...`), Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєРё РёР·РІР»РµРєР°РµС‚ host Рё mask;
- РµСЃР»Рё РІРІРµРґС‘РЅ С‚РѕР»СЊРєРѕ host/base URL, РґРѕРїРѕР»РЅРёС‚РµР»СЊРЅРѕ СЃРїСЂР°С€РёРІР°РµС‚ mask (РїРѕ СѓРјРѕР»С‡Р°РЅРёСЋ `/endpoints?service=<db>.pg:bouncer`).
- дополнительно не спрашивает lookup-user/auth_query: локальный proxy не терминирует auth и не подменяет реальные DB credentials пользователя.

`db add <service>`:
- СЃРїСЂР°С€РёРІР°РµС‚ С‚РѕР»СЊРєРѕ РёРјСЏ Р±Р°Р·С‹ (РµСЃР»Рё РЅРµ РїРµСЂРµРґР°РЅРѕ Р°СЂРіСѓРјРµРЅС‚РѕРј);
- РїРѕРґСЃС‚Р°РІР»СЏРµС‚ РµРіРѕ РІ `warden host + mask`, Р·Р°РїСЂР°С€РёРІР°РµС‚ endpoint РёР· Warden;
- РґРµР»Р°РµС‚ TCP-check endpoint;
- СЃРѕС…СЂР°РЅСЏРµС‚ Р±Р°Р·Сѓ РІ СЃРїРёСЃРѕРє С‚РѕР»СЊРєРѕ РїСЂРё СѓСЃРїРµС€РЅРѕР№ РїСЂРѕРІРµСЂРєРµ;
- обновляет/поднимает local proxy, чтобы база сразу была доступна по тому же `localhost:<port>`.

РџРѕРґРєР»СЋС‡РµРЅРёРµ РёР· IDE:
- РІ РєР»РёРµРЅС‚Рµ РЅСѓР¶РЅРѕ СѓРєР°Р·С‹РІР°С‚СЊ СЂРµР°Р»СЊРЅС‹Рµ username/password РѕС‚ Р±Р°Р·С‹;
- локальный слой прозрачно прокидывает PostgreSQL traffic в выбранный upstream и не требует отдельного lookup-user.

`db start`:
- РёСЃРїРѕР»СЊР·СѓРµС‚ СѓР¶Рµ РЅР°СЃС‚СЂРѕРµРЅРЅС‹Рµ Warden host/mask Рё РґРѕР±Р°РІР»РµРЅРЅС‹Рµ Р±Р°Р·С‹;
- подтягивает актуальные endpoint-ы из Warden и запускает local proxy на одном порту (по умолчанию `15432`);
- РЅСѓР¶РµРЅ РєР°Рє РІРѕСЃСЃС‚Р°РЅРѕРІРёС‚РµР»СЊРЅС‹Р№ Р·Р°РїСѓСЃРє РїРѕСЃР»Рµ РїРµСЂРµР·Р°РіСЂСѓР·РєРё WSL РёР»Рё `db stop`.

РџСЂРѕРІРµСЂРєР° Рё РѕСЃС‚Р°РЅРѕРІРєР°:

```bash
wslbridge db status
wslbridge db stop
```

Р Р°Р±РѕС‚Р° СЃРѕ СЃРїРёСЃРєРѕРј Р±Р°Р·:

```bash
wslbridge db add chatapi-ng
wslbridge db add bozon-saturn
wslbridge db status
wslbridge db start     # РІРѕСЃСЃС‚Р°РЅРѕРІРёС‚СЊ Р·Р°РїСѓСЃРє РїРѕСЃР»Рµ СЂРµСЃС‚Р°СЂС‚Р° WSL
```

`db add <service>` добавляет сервис только если Warden вернул endpoint и TCP-коннект до него успешен. Для IDE используйте `sslmode=disable` или `sslmode=prefer`: локальный proxy отвечает `N` на PostgreSQL `SSLRequest`.

---

## Р”РёР°РіРЅРѕСЃС‚РёРєР°

- РџРѕСЃРјРѕС‚СЂРµС‚СЊ СЃС‚Р°С‚СѓСЃ: `wslbridge status`
- РћСЃС‚Р°РЅРѕРІРёС‚СЊ Рё РІРµСЂРЅСѓС‚СЊ РјР°СЂС€СЂСѓС‚С‹: `wslbridge stop`
- РџСЂРѕРІРµСЂРёС‚СЊ РјР°СЂС€СЂСѓС‚ РїРѕ СѓРјРѕР»С‡Р°РЅРёСЋ: `ip route show default`
- РџСЂРѕРІРµСЂРёС‚СЊ, С‡С‚Рѕ `tun2socks` Р¶РёРІ: `kill -0 $(cat ~/.local/state/wslbridge/tun2socks.pid)`
- Р•СЃР»Рё С‚СЂР°С„РёРє РЅРµ РёРґС‘С‚, Р·Р°РіР»СЏРЅРёС‚Рµ РІ `/tmp/tun2socks.log` Рё СѓР±РµРґРёС‚РµСЃСЊ, С‡С‚Рѕ SOCKS-РїСЂРѕРєСЃРё РґРѕСЃС‚СѓРїРµРЅ РёР· WSL.

РџСЂРёРјРµСЂ:

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

