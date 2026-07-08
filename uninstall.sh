#!/usr/bin/env bash
# Удаление unissh (конфиг и пароли в Keychain остаются)
set -euo pipefail

rm -f "${HOME}/.local/bin/unissh"
echo "✓ Бинарник удалён."
echo ""
echo "Остались (удалите вручную, если нужно):"
echo "  ~/.config/unissh/            — конфиг, known_hosts"
echo "  ~/.ssh/unissh_ed25519(.pub)  — SSH-ключ"
echo "  ~/.ssh/unissh_config         — экспорт ssh-конфига (+ строка Include в ~/.ssh/config)"
echo "  Keychain: пароли с сервисом 'unissh' (Keychain Access → поиск 'unissh')"
