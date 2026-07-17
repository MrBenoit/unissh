#!/usr/bin/env bash
# Запись demo.gif на ВЫДУМАННЫХ данных.
#
# Зачем скрипт, а не просто `vhs demo.tape`: unissh читает конфиг из
# ~/.config/unissh, поэтому голая запись покажет НАСТОЯЩИЕ серверы автора —
# их IP, логины и имена. Это карта чужой инфраструктуры в публичной гифке.
#
# Что делает скрипт:
#   1. подменяет $HOME на временный с фиктивным конфигом и одноразовым ключом;
#   2. поднимает loopback-псевдонимы 203.0.113.x — это TEST-NET-3, диапазон,
#      зарезервированный RFC 5737 ровно для документации (нужен sudo);
#   3. запускает demoserver — он говорит по SSH и отдаёт правдоподобные /proc,
#      поэтому в кадре видно живые шкалы CPU/RAM/диска, а не «loading…».
#
# Требуется: vhs (brew install vhs), sudo (только для псевдонимов), go.
# Запуск: make demo
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

command -v vhs >/dev/null 2>&1 || { echo "нужен vhs: brew install vhs"; exit 1; }

# Адреса серверов в демо. Идеальный вариант — TEST-NET-3 (203.0.113.0/24):
# RFC 5737 резервирует его ровно под документацию, и в кадре он выглядит как
# настоящий парк. Для этого адреса надо поднять на loopback, а это sudo.
# Без sudo откатываемся на 127.0.0.1 с разными портами: демо запишется, но
# будет видно, что это песочница.
IPS=(203.0.113.10 203.0.113.11 203.0.113.20 203.0.113.30 203.0.113.31)
PORTS=(2222 2222 2222 2222 2222) # порт в списке серверов не отображается
USE_ALIASES=1
if ! command -v sudo >/dev/null 2>&1; then
    USE_ALIASES=0
elif ! sudo -n true 2>/dev/null; then
    # Кэша sudo нет. Если терминал есть — пусть спросит пароль; проверять только
    # `sudo -n` нельзя: тогда скрипт молча уходит на запасной путь, ни о чём не
    # спросив, и демо каждый раз пишется на 127.0.0.1.
    if [ -t 0 ]; then
        echo "Для реалистичных IP в демо нужен sudo (loopback-псевдонимы)."
        sudo -v || USE_ALIASES=0
    else
        USE_ALIASES=0
    fi
fi
if [ "$USE_ALIASES" = 0 ]; then
    echo "⚠  Пишу на 127.0.0.1 — в кадре будет видно, что это песочница."
    echo "   Для реалистичных IP запусти 'make demo' в терминале и введи пароль."
    IPS=(127.0.0.1 127.0.0.1 127.0.0.1 127.0.0.1 127.0.0.1)
    PORTS=(2201 2202 2203 2204 2205)
fi

DEMO_HOME="$(mktemp -d)"
cleanup() {
    [ -n "${SRV_PID:-}" ] && kill "$SRV_PID" 2>/dev/null || true
    if [ "$USE_ALIASES" = 1 ]; then
        for ip in "${IPS[@]}"; do sudo ifconfig lo0 -alias "$ip" 2>/dev/null || true; done
    fi
    rm -rf "$DEMO_HOME"
}
trap cleanup EXIT

if [ "$USE_ALIASES" = 1 ]; then
    echo "Поднимаю loopback-псевдонимы 203.0.113.x…"
    for ip in "${IPS[@]}"; do sudo ifconfig lo0 alias "$ip" up; done
fi

# --- фиктивный конфиг -------------------------------------------------------
# Язык en: гифка идёт в англоязычный README.
mkdir -p "$DEMO_HOME/.config/unissh" "$DEMO_HOME/.ssh"
cat > "$DEMO_HOME/.config/unissh/servers.json" <<JSON
{
  "version": 1,
  "settings": {
    "default_user": "deploy",
    "ssh_key": "~/.ssh/unissh_ed25519",
    "status_timeout": 1.5,
    "language": "en",
    "theme": "blue"
  },
  "folders": ["prod", "staging"],
  "servers": [
    {"id":"1a2b3c4d","name":"web-1","host":"${IPS[0]}","port":${PORTS[0]},"user":"deploy","auth":"key","key":"~/.ssh/unissh_ed25519","folder":"prod","favorite":true,"notes":"nginx + app"},
    {"id":"2b3c4d5e","name":"web-2","host":"${IPS[1]}","port":${PORTS[1]},"user":"deploy","auth":"key","key":"~/.ssh/unissh_ed25519","folder":"prod","notes":"nginx + app"},
    {"id":"3c4d5e6f","name":"db-primary","host":"${IPS[2]}","port":${PORTS[2]},"user":"deploy","auth":"key","key":"~/.ssh/unissh_ed25519","folder":"prod","notes":"postgres 16"},
    {"id":"4d5e6f70","name":"staging-web","host":"${IPS[3]}","port":${PORTS[3]},"user":"deploy","auth":"key","key":"~/.ssh/unissh_ed25519","folder":"staging"},
    {"id":"5e6f7081","name":"ci-runner","host":"${IPS[4]}","port":${PORTS[4]},"user":"deploy","auth":"key","key":"~/.ssh/unissh_ed25519","folder":"staging"}
  ]
}
JSON

# Ключ должен быть НАСТОЯЩИМ: unissh грузит его перед входом, на пустом файле
# разбор упадёт и статистика покажет ошибку вместо шкал.
ssh-keygen -t ed25519 -N "" -C "demo@unissh" -f "$DEMO_HOME/.ssh/unissh_ed25519" -q

# --- демо-серверы: отвечают и на TCP-пробу (зелёный «онлайн»), и на SSH ------
go build -o "$DEMO_HOME/bin/demoserver" ./scripts/demoserver
LOADS=("42,31,24" "17,44,29" "63,72,58" "8,22,15" "91,66,37")
SPECS=()
for i in "${!IPS[@]}"; do SPECS+=("${IPS[$i]}:${PORTS[$i]}=${LOADS[$i]}"); done
"$DEMO_HOME/bin/demoserver" "${SPECS[@]}" &
SRV_PID=$!
sleep 0.7

# --- запись -----------------------------------------------------------------
go build -ldflags "-s -w -X main.version=v1.0.1" -o "$DEMO_HOME/bin/unissh" .
echo "Записываю demo.gif…"
HOME="$DEMO_HOME" PATH="$DEMO_HOME/bin:$PATH" vhs demo.tape

echo "✓ demo.gif готов — на фиктивных серверах, настоящие в кадр попасть не могли."
