package core

// Разбор строк `free -b`, `df -P -k /` и /proc/loadavg в числа. Одни и те же
// строки нужны и панели статистики, и дашборду здоровья парка, поэтому парсинг
// живёт здесь, а не в отрисовке.

import (
	"strconv"
	"strings"
)

// MemUsage разбирает вторую строку `free -b` («Mem: total used …») и возвращает
// занято/всего в байтах. ok=false, если строка не похожа на ожидаемую.
func MemUsage(line string) (used, total float64, ok bool) {
	f := strings.Fields(line)
	if len(f) < 3 {
		return 0, 0, false
	}
	total, errT := strconv.ParseFloat(f[1], 64)
	used, errU := strconv.ParseFloat(f[2], 64)
	if errT != nil || errU != nil || total <= 0 {
		return 0, 0, false
	}
	return used, total, true
}

// DiskUsage разбирает строку `df -P -k /` и возвращает занято/всего в БАЙТАХ:
// df печатает килобайты, а показываем мы байты.
func DiskUsage(line string) (used, total float64, ok bool) {
	f := strings.Fields(line)
	if len(f) < 4 {
		return 0, 0, false
	}
	total, errT := strconv.ParseFloat(f[1], 64)
	used, errU := strconv.ParseFloat(f[2], 64)
	if errT != nil || errU != nil || total <= 0 {
		return 0, 0, false
	}
	return used * 1024, total * 1024, true
}

// LoadFields возвращает первые три числа /proc/loadavg одной строкой
// («0.10 0.20 0.30»). ok=false, если чисел меньше трёх.
func LoadFields(line string) (string, bool) {
	f := strings.Fields(line)
	if len(f) < 3 {
		return "", false
	}
	return strings.Join(f[:3], " "), true
}

// LoadPerCPU — нагрузка за минуту, нормированная на число ядер: 1.0 означает
// «процессоры заняты ровно под завязку». Именно её удобно сравнивать между
// машинами с разным числом ядер.
func LoadPerCPU(line string, ncpu int) (float64, bool) {
	f := strings.Fields(line)
	if len(f) == 0 || ncpu < 1 {
		return 0, false
	}
	l1, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0, false
	}
	return l1 / float64(ncpu), true
}

// Frac возвращает долю used/total, обрезанную в [0, 1]. total<=0 → 0.
func Frac(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	f := used / total
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// OneLine сжимает многострочный текст в одну строку: переводы строк заменяются
// на «⏎», хвост обрезается по РУНАМ (обрезка по байтам разрубила бы кириллицу).
// Нужен там, где произвольный текст должен уместиться в ячейку таблицы.
func OneLine(s string, max int) string {
	s = strings.Join(strings.Fields(strings.NewReplacer("\r", "\n", "\n", " ⏎ ").Replace(s)), " ")
	r := []rune(s)
	if max > 1 && len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}
