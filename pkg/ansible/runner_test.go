package ansible

import (
	"testing"
)

func TestBuildArgs_BasicPlaybook(t *testing.T) {
	args := BuildArgs("deploy.yml", "/tmp/inv.ini", nil)

	expected := []string{"deploy.yml", "--inventory", "/tmp/inv.ini"}
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

	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d: %v", len(args), args)
	}
	if args[3] != "--tags" || args[4] != "deploy" {
		t.Errorf("expected --tags deploy in args: %v", args)
	}
}

type mockProvider struct {
	password string
	err      error
}

func (m *mockProvider) Fetch() (string, error) {
	return m.password, m.err
}

func TestRunner_BuildCmd_SetsEnvVar(t *testing.T) {
	provider := &mockProvider{password: "testpass123"}
	r := NewRunner(provider)

	cmd, err := r.BuildCmd("testpass123", "echo", "/tmp/inv.ini", []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, env := range cmd.Env {
		if env == "ANSIBLE_BECOME_PASS=testpass123" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ANSIBLE_BECOME_PASS=testpass123 in env, got: %v", cmd.Env)
	}
}
