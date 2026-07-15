package core

// Сборка shell-скрипта автоочистки буфера обмена. Команды буфера у каждой ОС
// свои (pbcopy/wl-copy/xclip/xsel), поэтому сюда они приходят параметрами —
// здесь только правильная склейка.

import "fmt"

// ClipClearScript строит sh-скрипт, который через ttl секунд сравнивает SHA-256
// текущего содержимого буфера с ожидаемым и очищает буфер только при совпадении:
// если пользователь успел скопировать что-то своё, мы это не затираем.
//
// САМ СЕКРЕТ В СКРИПТ НЕ ПОПАДАЕТ — только его хеш. Скрипт уходит в argv
// фонового `sh -c`, и там секрет виден был бы любому через `ps`.
//
//	pasteCmd — команда, печатающая содержимое буфера (pbpaste, wl-paste …)
//	hashCmd  — команда, печатающая SHA-256 своего stdin (shasum -a 256, sha256sum)
//	clearCmd — команда, очищающая буфер (printf "" | pbcopy, wl-copy --clear …)
func ClipClearScript(pasteCmd, hashCmd, clearCmd, sum string, ttl int) string {
	return fmt.Sprintf(
		`sleep %d; if [ "$(%s | %s | cut -d" " -f1)" = "%s" ]; then %s; fi`,
		ttl, pasteCmd, hashCmd, sum, clearCmd)
}
