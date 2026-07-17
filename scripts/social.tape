# VHS-сценарий для .github/social-preview.png — картинки 1280×640, которую
# GitHub показывает вместо серого прямоугольника при шаринге ссылки.
#
# НЕ запускай напрямую: нужен подготовленный $HOME с фиктивными серверами,
# иначе в превью попадут ТВОИ настоящие. Запускай `make social`.
#
# Название и описание проекта подставит сам GitHub (og:title/og:description),
# поэтому здесь только интерфейс — он и должен продавать.

# VHS требует Output, хотя нам нужен только Screenshot — пишем во временный
# файл, его убирает цель `make social`.
Output .social-tmp.gif

Set Width 1280
Set Height 640
Set FontSize 16
Set Theme "Dracula"
Set Padding 20

Sleep 600ms
Type "unissh"
Enter
Sleep 1200ms

# Курсор на сервер и ждём: тик опроса 2 с, а CPU% считается по дельте двух
# снимков — без этой паузы в превью будет «loading…» вместо шкал.
Down
Sleep 7s

Screenshot .github/social-preview.png
