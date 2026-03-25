package ansible

import (
	"testing"

	"github.com/geraldcsoftware/playbook/pkg/credentials"
)

func TestBuildArgs_BasicPlaybook(t *testing.T) {
	args := BuildArgs("deploy.yml", "/tmp/inv.ini", nil)

	expected := []string{"deploy.yml", "--inventory", "/tmp/inv.ini", "--extra-vars", "ansible_become_pass={{ lookup('env', 'ANSIBLE_BECOME_PASS') }}"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d]: expected '%s', got '%s'", i, expected[i], a)
		}
	}
}

func TestBuildArgs_WithExtraArgs(t *testing.T) {
	args := BuildArgs("deploy.yml", "/tmp/inv.ini", []string{"--tags", "deploy", "--limit", "web"})

	found := false
	for i, a := range args {
		if a == "--tags" && i+1 < len(args) && args[i+1] == "deploy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --tags deploy in args: %v", args)
	}
}

func TestNewRunner(t *testing.T) {
	provider := &credentials.AACProvider{BinaryPath: "/usr/bin/true"}
	r := NewRunner(provider)
	if r == nil {
		t.Fatal("expected non-nil runner")
	}
}
