package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEscapeHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"db-prod.eus.v.co.zw", "db_prod_eus_v_co_zw"},
		{"simple", "simple"},
		{"host.with.dots", "host_with_dots"},
	}
	for _, tc := range tests {
		got := EscapeHostname(tc.input)
		if got != tc.expected {
			t.Errorf("EscapeHostname(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestKeyPath(t *testing.T) {
	path := KeyPath("ed25519", "db-prod.eus.v.co.zw")
	if !strings.Contains(path, "id_ed25519_db_prod_eus_v_co_zw") {
		t.Errorf("unexpected key path: %s", path)
	}
}

func TestBuildSSHConfigEntry(t *testing.T) {
	entry := BuildSSHConfigEntry("db-prod.eus.v.co.zw", "gchifanzwa", "~/.ssh/id_ed25519_db_prod", 22)
	if !strings.Contains(entry, "Host db-prod.eus.v.co.zw") {
		t.Error("expected Host line in entry")
	}
	if !strings.Contains(entry, "User gchifanzwa") {
		t.Error("expected User line in entry")
	}
	if !strings.Contains(entry, "IdentityFile ~/.ssh/id_ed25519_db_prod") {
		t.Error("expected IdentityFile line in entry")
	}
}

func TestAppendSSHConfigEntry(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	os.WriteFile(configPath, []byte("# existing config\n"), 0644)

	entry := BuildSSHConfigEntry("test.example.com", "user", "~/.ssh/id_ed25519_test", 22)
	err := AppendSSHConfigEntry(configPath, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	content := string(data)
	if !strings.Contains(content, "# existing config") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, "Host test.example.com") {
		t.Error("new entry should be appended")
	}
}
