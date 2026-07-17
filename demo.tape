# VHS-скрипт для записи demo.gif: https://github.com/charmbracelet/vhs
#
# НЕ запускай `vhs demo.tape` напрямую: unissh покажет ТВОИ настоящие серверы —
# их IP, логины и имена уедут в публичную гифку.
# Запускай `make demo` — там подменяется $HOME на временный с фиктивным
# конфигом и поднимаются демо-серверы (см. scripts/record-demo.sh).
#
# Про паузы: панель статистики опрашивает сервер раз в 2 с, а процент CPU
# считается по дельте ДВУХ снимков. Поэтому на сервере надо задержаться
# ~5 с, иначе в кадре будет «loading…» вместо шкал.

Output demo.gif

Set FontSize 18
Set Width 1200
Set Height 750
Set Theme "Dracula"
Set Padding 20

Sleep 800ms
Type "unissh"
Enter
Sleep 1200ms

# Курсор на первый сервер — справа действия и живая статистика.
# 7 с не случайны: тик опроса 2 с, первый снимок даёт RAM/диск/load,
# второй (ещё +2 с) — CPU% и скорость сети. Это главный кадр демо.
Down
Sleep 7s

# переключаемся на другой сервер: статистика едет за курсором
Down
Sleep 3500ms

# поиск (фильтр ищет по имени, хосту и логину — не по папкам)
Type "/"
Sleep 500ms
Type "web"
Sleep 1800ms
Escape
Sleep 800ms

# мультивыбор: space отмечает серверы, правая панель → групповые действия
Space
Sleep 700ms
Down
Sleep 400ms
Space
Sleep 1800ms
Escape
Sleep 800ms

# справка по клавишам
Type "?"
Sleep 2200ms
Escape
Sleep 600ms

Type "q"
Sleep 800ms
