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
    elif command -v apt >/dev/null 2>&1; then
        echo "Установите: ${BOLD}sudo apt install golang-go${RESET} (нужен Go ≥ 1.26)"
        echo "или свежий Go с https://go.dev/dl/ — и запустите скрипт снова."
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

# --- PATH: дописываем в rc-файл ТЕКУЩЕГО шелла, а не в ~/.zshrc вслепую ---
shell_rc() {
    case "$(basename "${SHELL:-}")" in
        zsh)  echo "${ZDOTDIR:-$HOME}/.zshrc" ;;
        bash) [ -f "${HOME}/.bash_profile" ] && echo "${HOME}/.bash_profile" || echo "${HOME}/.bashrc" ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        *)    echo "${HOME}/.profile" ;;
    esac
}

case ":$PATH:" in
    *":${BIN_DIR}:"*) ;;
    *)
        SHELL_RC="$(shell_rc)"
        mkdir -p "$(dirname "$SHELL_RC")"
        if grep -qs '\.local/bin' "$SHELL_RC" 2>/dev/null; then
            # строка в rc уже есть, но текущая сессия её не подхватила
            echo "${YELLOW}⚠${RESET} ${BIN_DIR} уже прописан в ${SHELL_RC}, но не активен в этой сессии."
            echo "  Перезапустите терминал или выполните: ${BOLD}source ${SHELL_RC}${RESET}"
        else
            if [ "$(basename "${SHELL:-}")" = "fish" ]; then
                printf '\n# добавлено установщиком unissh\nfish_add_path "$HOME/.local/bin"\n' >> "$SHELL_RC"
            else
                printf '\n# добавлено установщиком unissh\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$SHELL_RC"
            fi
            echo "${YELLOW}⚠${RESET} ${BIN_DIR} добавлен в PATH через ${SHELL_RC}."
            echo "  Перезапустите терминал или выполните: ${BOLD}source ${SHELL_RC}${RESET}"
        fi
        ;;
esac

# --- необязательные, но полезные зависимости ---
missing=()
command -v expect >/dev/null 2>&1 || missing+=("expect — авто-ввод пароля при подключении")

if [ "$(uname -s)" = "Linux" ]; then
    if ! command -v wl-copy >/dev/null 2>&1 && ! command -v xclip >/dev/null 2>&1 && ! command -v xsel >/dev/null 2>&1; then
        missing+=("wl-clipboard (Wayland) или xclip (X11) — копирование паролей в буфер")
    fi
    command -v secret-tool >/dev/null 2>&1 || \
        missing+=("libsecret-tools — пароли в системной ключнице вместо файла 0600")
fi

if [ ${#missing[@]} -gt 0 ]; then
    echo ""
    echo "${YELLOW}Не найдены необязательные компоненты:${RESET}"
    for m in "${missing[@]}"; do echo "  · ${m}"; done
    if command -v apt >/dev/null 2>&1; then
        echo ""
        echo "  ${DIM}sudo apt install expect wl-clipboard libsecret-tools${RESET}"
    elif command -v dnf >/dev/null 2>&1; then
        echo ""
        echo "  ${DIM}sudo dnf install expect wl-clipboard libsecret${RESET}"
    fi
    echo "${DIM}  unissh работает и без них — просто с меньшим комфортом.${RESET}"
fi

echo ""
echo "${GREEN}Готово!${RESET} Запустите: ${BOLD}unissh${RESET}"
