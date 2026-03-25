package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

const testSSHConfig = `Host db-prod.eus.v.co.zw
    HostName db-prod.eus.v.co.zw
    User gchifanzwa
    IdentityFile ~/.ssh/id_rsa_db_prod
    Port 22

Host web-01.eus.v.co.zw
    HostName web-01.eus.v.co.zw
    User deploy
    IdentityFile ~/.ssh/id_ed25519_web01
    Port 2222

Host *
    ServerAliveInterval 60
`

func TestParseConfig(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config")
	os.WriteFile(f, []byte(testSSHConfig), 0644)

	hosts, err := ParseConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	h := hosts[0]
	if h.Alias != "db-prod.eus.v.co.zw" {
		t.Errorf("expected alias db-prod.eus.v.co.zw, got %s", h.Alias)
	}
	if h.HostName != "db-prod.eus.v.co.zw" {
		t.Errorf("expected hostname db-prod.eus.v.co.zw, got %s", h.HostName)
	}
	if h.User != "gchifanzwa" {
		t.Errorf("expected user gchifanzwa, got %s", h.User)
	}
	if h.IdentityFile != "~/.ssh/id_rsa_db_prod" {
		t.Errorf("expected identity file ~/.ssh/id_rsa_db_prod, got %s", h.IdentityFile)
	}
	if h.Port != 22 {
		t.Errorf("expected port 22, got %d", h.Port)
	}

	h2 := hosts[1]
	if h2.Port != 2222 {
		t.Errorf("expected port 2222, got %d", h2.Port)
	}
	if h2.User != "deploy" {
		t.Errorf("expected user deploy, got %s", h2.User)
	}
}

func TestParseConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config")
	os.WriteFile(f, []byte(""), 0644)

	hosts, err := ParseConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseConfig_MissingFile(t *testing.T) {
	_, err := ParseConfig("/nonexistent/config")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
