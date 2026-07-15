package core

// Подготовка путей для scp. Удалённый путь попадает в шелл на той стороне, а
// локальный разбирается самим scp — правила у них разные, поэтому и функции две.

import (
	"regexp"
	"strings"
)

// safeRemotePath — символы, которые удалённый шелл трактует буквально.
// Всё остальное (пробелы, кавычки, $, ;, звёздочки) обязано быть закавычено.
var safeRemotePath = regexp.MustCompile(`^[A-Za-z0-9._/@%+:,=-]+$`)

// quoteIfNeeded закавычивает путь, только если в нём есть что-то опасное:
// лишние кавычки в argv читаются человеком хуже, чем их отсутствие.
func quoteIfNeeded(p string) string {
	if safeRemotePath.MatchString(p) {
		return p
	}
	return ShellQuote(p)
}

// RemotePathArg готовит удалённый путь для scp (часть после «user@host:»).
// Пустой путь означает домашний каталог. Ведущая тильда НЕ экранируется —
// иначе удалённый шелл не раскроет её и файл уедет в каталог с именем «~».
// Всё, что за ней, экранируется: пробел или «;» в пути не должны превращаться
// в команду на сервере.
func RemotePathArg(p string) string {
	p = strings.TrimSpace(p)
	switch {
	case p == "":
		return "."
	case p == "~", p == "~/":
		return "~/"
	case strings.HasPrefix(p, "~/"):
		return "~/" + quoteIfNeeded(p[2:])
	}
	return quoteIfNeeded(p)
}

// LocalPathArg готовит локальный путь для scp. Двоеточие в первом сегменте scp
// принимает за «host:path», поэтому такому пути нужен явный префикс «./».
func LocalPathArg(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "."
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") {
		return p
	}
	head, _, _ := strings.Cut(p, "/")
	if strings.Contains(head, ":") {
		return "./" + p
	}
	return p
}
