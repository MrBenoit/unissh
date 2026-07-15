package app

import (
	"reflect"
	"regexp"
	"testing"
)

// formatVerb ловит плейсхолдеры вида %s, %d, %.1f, %v.
var formatVerb = regexp.MustCompile(`%[#+\- 0]*[\d.]*[a-zA-Z]`)

// stringFields собирает все строковые поля словаря: имя → значение.
func stringFields(l L) map[string]string {
	out := map[string]string{}
	v := reflect.ValueOf(l)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type.Kind() == reflect.String {
			out[t.Field(i).Name] = v.Field(i).String()
		}
	}
	return out
}

// TestI18nNoEmptyStrings: пустое поле — это пустая подпись в интерфейсе.
// Именно так «shift+↑↓» долго висел в справке без текста: поле объявили,
// значение задать забыли, и компилятор промолчал.
func TestI18nNoEmptyStrings(t *testing.T) {
	for lang, dict := range map[string]L{"ru": trRU, "en": trEN} {
		for name, val := range stringFields(dict) {
			if val == "" {
				t.Errorf("%s: строка %s не задана", lang, name)
			}
		}
	}
}

// TestI18nSameFormatVerbs: у русского и английского вариантов одной строки
// должны совпадать плейсхолдеры — иначе fmt.Sprintf напечатает %!d(MISSING).
func TestI18nSameFormatVerbs(t *testing.T) {
	ru, en := stringFields(trRU), stringFields(trEN)
	for name, ruVal := range ru {
		enVal := en[name]
		a, b := formatVerb.FindAllString(ruVal, -1), formatVerb.FindAllString(enVal, -1)
		if len(a) != len(b) {
			t.Errorf("%s: ru=%v en=%v (ru=%q en=%q)", name, a, b, ruVal, enVal)
			continue
		}
		for i := range a {
			if a[i] != b[i] {
				t.Errorf("%s: плейсхолдер #%d — ru %s, en %s", name, i, a[i], b[i])
			}
		}
	}
}

// TestI18nBytesUnits: единицы размера заданы в обоих языках (их использует humanBytes).
func TestI18nBytesUnits(t *testing.T) {
	for lang, dict := range map[string]L{"ru": trRU, "en": trEN} {
		if len(dict.BytesUnits) < 4 {
			t.Errorf("%s: BytesUnits = %v", lang, dict.BytesUnits)
		}
	}
}
