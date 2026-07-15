#!/usr/bin/env bash
# Удаление unissh (конфиг и сохранённые пароли остаются)
set -euo pipefail

rm -f "${HOME}/.local/bin/unissh"
echo "✓ Бинарник удалён."
echo ""
echo "Остались (удалите вручную, если нужно):"
echo "  ~/.config/unissh/            — конфиг, known_hosts, журнал, бэкапы"
echo "  ~/.ssh/unissh_ed25519(.pub)  — SSH-ключ"
echo "  ~/.ssh/unissh_config         — экспорт ssh-конфига (+ строка Include в ~/.ssh/config)"

if [ "$(uname -s)" = "Darwin" ]; then
    echo "  Keychain: пароли с сервисом 'unissh' (Keychain Access → поиск 'unissh')"
else
    if command -v secret-tool >/dev/null 2>&1; then
        echo "  Ключница: пароли с service=unissh (secret-tool clear service unissh account <id>/<user>)"
    fi
    echo "  ~/.config/unissh/secrets.json — пароли, если ключницы не было"
fi
