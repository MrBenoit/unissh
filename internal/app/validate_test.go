package app

import "testing"

func TestValidateHost(t *testing.T) {
	ok := []string{"example.com", "10.0.0.1", "srv-1.internal", "2a03:b0c0::1"}
	bad := []string{"", "  ", "has space", "tab\ttab"}
	for _, s := range ok {
		if err := validateHost(s); err != nil {
			t.Errorf("validateHost(%q) неожиданно ошибка: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateHost(s); err == nil {
			t.Errorf("validateHost(%q) должно быть ошибкой", s)
		}
	}
}

func TestValidatePort(t *testing.T) {
	ok := []string{"1", "22", "65535", " 2222 "}
	bad := []string{"0", "-1", "65536", "abc", "", "22.5"}
	for _, s := range ok {
		if err := validatePort(s); err != nil {
			t.Errorf("validatePort(%q) неожиданно ошибка: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validatePort(s); err == nil {
			t.Errorf("validatePort(%q) должно быть ошибкой", s)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	ok := []string{"root", "deploy", "_x", "a1-b_c", "user123"}
	bad := []string{"", "1user", "Root", "has space", "toolongusernaaaaaaaaaaaaaaaaaaaaame"}
	for _, s := range ok {
		if err := validateUsername(s); err != nil {
			t.Errorf("validateUsername(%q) неожиданно ошибка: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateUsername(s); err == nil {
			t.Errorf("validateUsername(%q) должно быть ошибкой", s)
		}
	}
}

func TestValidateName(t *testing.T) {
	st := &Store{Config: Config{Servers: []*Server{{ID: "1", Name: "taken"}}}}
	v := validateName(st, "")
	if err := v("good-name_1.2"); err != nil {
		t.Errorf("корректное имя отклонено: %v", err)
	}
	if err := v(""); err == nil {
		t.Error("пустое имя должно быть ошибкой")
	}
	if err := v("bad name!"); err == nil {
		t.Error("имя с недопустимыми символами должно быть ошибкой")
	}
	if err := v("taken"); err == nil {
		t.Error("занятое имя должно быть ошибкой")
	}
	// то же имя, но исключаем его же ID — допустимо (редактирование)
	if err := validateName(st, "1")("taken"); err != nil {
		t.Errorf("своё имя при редактировании должно проходить: %v", err)
	}
}
