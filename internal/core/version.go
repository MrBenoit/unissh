package core

// Выбор версии, которую показывать пользователю.

// DevVersion — заглушка, когда версия ниоткуда не пришла (обычный `go build`).
const DevVersion = "dev"

// develBuildInfo — что runtime/debug сообщает для локальной сборки. Это не
// версия, а признак её отсутствия.
const develBuildInfo = "(devel)"

// ResolveVersion выбирает версию для вывода: приоритет у линкера, запасной
// вариант — информация о сборке.
//
//	ldflags   — значение из -X main.version=… (его ставят goreleaser и `make build`)
//	buildInfo — версия модуля из runtime/debug (её Go проставляет сам при
//	            `go install module@version`)
//
// Зачем запасной вариант: `go install` НЕ применяет ldflags. Без него бинарник,
// установленный рекомендованным в README способом, рапортовал бы «dev» вместо
// «v1.0.0».
func ResolveVersion(ldflags, buildInfo string) string {
	if ldflags != "" && ldflags != DevVersion {
		return ldflags
	}
	if buildInfo != "" && buildInfo != develBuildInfo {
		return buildInfo
	}
	return DevVersion
}
