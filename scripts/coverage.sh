#!/usr/bin/env sh
# Гейт покрытия ядра логики.
#
# Покрытие измеряется по пакету internal/core (чистая логика без TUI/SSH/OS).
# Если оно ниже COVER_MIN (по умолчанию 90) — сборка и релиз запрещены.
# Тонкий интерактивный слой internal/app в порог не входит: он проверяется
# полным прогоном `go test ./...` (зелёные тесты), но не метрикой покрытия.
set -e

MIN="${COVER_MIN:-90}"
PKG="${COVER_PKG:-./internal/core/...}"

go test -coverprofile=coverage.out "$PKG" >/dev/null
total="$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"

echo "core coverage: ${total}% (min ${MIN}%)"
awk "BEGIN{ exit !(${total} >= ${MIN}) }" || {
	echo "ОШИБКА: покрытие ядра ${total}% ниже требуемых ${MIN}% — сборка/релиз запрещены." >&2
	exit 1
}
