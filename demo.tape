# VHS-скрипт для записи demo.gif: https://github.com/charmbracelet/vhs
# Запись:  vhs demo.tape   (нужен установленный unissh в PATH)

Output demo.gif

Set FontSize 18
Set Width 1200
Set Height 750
Set Theme "Dracula"
Set Padding 20

Sleep 1s
Type "unissh"
Enter
Sleep 2s

# навигация по списку серверов
Down
Sleep 800ms
Down
Sleep 800ms
Up
Sleep 800ms

# поиск
Type "/"
Sleep 500ms
Type "prod"
Sleep 1s
Escape
Sleep 800ms

# помощь
Type "?"
Sleep 2s
Escape
Sleep 800ms

# выход
Type "q"
Sleep 1s
