#!/usr/bin/env bash
# Установка unissh: сборка из исходников и установка в ~/.local/bin
set -euo pipefail

BOLD=$'\033[1m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; RED=$'\033[31m'; DIM=$'\033[2m'; RESET=$'\033[0m'

SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${HOME}/.local/bin"

echo "${BOLD}unissh — установка${RESET}"

if ! command -v go >/dev/null 2>&1; then
    echo "${RED}Go не найден.${RESET}"
    if command -v brew >/dev/null 2>&1; then
        echo "Установите: ${BOLD}brew install go${RESET} и запустите скрипт снова."
    else
        echo "Установите Go с https://go.dev/dl/ и запустите скрипт снова."
    fi
    exit 1
fi

echo "${DIM}Собираю бинарник…${RESET}"
cd "$SRC_DIR"
go build -trimpath -ldflags="-s -w" -o unissh .

mkdir -p "$BIN_DIR"
cp unissh "$BIN_DIR/unissh"
chmod +x "$BIN_DIR/unissh"
echo "${GREEN}✓${RESET} Установлено: ${BOLD}${BIN_DIR}/unissh${RESET}"

case ":$PATH:" in
    *":${BIN_DIR}:"*) ;;
    *)
        SHELL_RC="${HOME}/.zshrc"
        if grep -qs '\.local/bin' "$SHELL_RC" 2>/dev/null; then
            # строка в rc уже есть, но текущая сессия её не подхватила
            echo "${YELLOW}⚠${RESET} ${BIN_DIR} уже прописан в ${SHELL_RC}, но не активен в этой сессии."
            echo "  Перезапустите терминал или выполните: ${BOLD}source ${SHELL_RC}${RESET}"
        else
            printf '\n# добавлено установщиком unissh\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$SHELL_RC"
            echo "${YELLOW}⚠${RESET} ${BIN_DIR} добавлен в PATH через ${SHELL_RC}."
            echo "  Перезапустите терминал или выполните: ${BOLD}source ${SHELL_RC}${RESET}"
        fi
        ;;
esac

echo ""
echo "${GREEN}Готово!${RESET} Запустите: ${BOLD}unissh${RESET}"
