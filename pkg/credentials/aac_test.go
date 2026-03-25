package credentials

import (
	"testing"
)

func newTestProvider() *AACProvider {
	return &AACProvider{BinaryPath: "/usr/bin/true"}
}

func TestAACProvider_Wrap_BuildsCorrectCommand(t *testing.T) {
	p := newTestProvider()

	cmd, err := p.Wrap(
		"test-item-id",
		"ansible-playbook",
		[]string{"deploy.yml", "--inventory", "/tmp/inv.ini"},
		map[string]string{"password": "ANSIBLE_BECOME_PASS"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Path == "" {
		t.Fatal("expected cmd.Path to be set")
	}

	args := cmd.Args
	found := false
	for i, a := range args {
		if a == "--id" && i+1 < len(args) && args[i+1] == "test-item-id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --id test-item-id in args: %v", args)
	}

	separatorIdx := -1
	for i, a := range args {
		if a == "--" {
			separatorIdx = i
			break
		}
	}
	if separatorIdx == -1 {
		t.Fatalf("expected -- separator in args: %v", args)
	}
	if args[separatorIdx+1] != "ansible-playbook" {
		t.Errorf("expected ansible-playbook after --, got %s", args[separatorIdx+1])
	}
}

func TestAACProvider_Wrap_MultipleEnvMappings(t *testing.T) {
	p := newTestProvider()

	cmd, err := p.Wrap(
		"item-123",
		"echo",
		[]string{"hello"},
		map[string]string{"password": "PASS", "username": "USER"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mapCount := 0
	for _, a := range cmd.Args {
		if a == "--map" {
			mapCount++
		}
	}
	if mapCount != 2 {
		t.Errorf("expected 2 --map flags, got %d. Args: %v", mapCount, cmd.Args)
	}
}

func TestAACProvider_Wrap_EmptyItemID(t *testing.T) {
	p := newTestProvider()
	_, err := p.Wrap("", "cmd", nil, map[string]string{"password": "PASS"})
	if err == nil {
		t.Error("expected error for empty item ID")
	}
}
