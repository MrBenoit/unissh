package app

import (
	"strings"
	"testing"
)

func TestSSHArgs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := &Server{User: "root", Host: "10.0.0.1", Port: 2222}
	args := strings.Join(sshArgs(s), " ")
	if !strings.Contains(args, "-p 2222") {
		t.Errorf("нет порта: %s", args)
	}
	if !strings.HasSuffix(args, "root@10.0.0.1") {
		t.Errorf("цель должна быть последней: %s", args)
	}
	if strings.Contains(args, "-i ") || strings.Contains(args, "-J ") {
		t.Errorf("без ключа и бастиона не должно быть -i/-J: %s", args)
	}
}

func TestSSHArgsKeyAndProxy(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	s := &Server{User: "deploy", Host: "target", Port: 22, Key: "~/.ssh/id", ProxyJump: "bastion:2200"}
	args := sshArgs(s)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-i /home/tester/.ssh/id") {
		t.Errorf("ключ должен раскрываться: %s", joined)
	}
	if !strings.Contains(joined, "IdentitiesOnly=yes") {
		t.Errorf("нет IdentitiesOnly: %s", joined)
	}
	if !strings.Contains(joined, "-J bastion:2200") {
		t.Errorf("нет ProxyJump: %s", joined)
	}
	if args[len(args)-1] != "deploy@target" {
		t.Errorf("цель должна быть последней: %v", args)
	}
}
