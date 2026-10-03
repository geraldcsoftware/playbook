package ssh

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const testSSHConfig = `Host db-prod.eus.v.co.zw
    HostName db-prod.eus.v.co.zw
    User deploy
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
	if len(h.Aliases) != 1 || h.Aliases[0] != "db-prod.eus.v.co.zw" {
		t.Errorf("expected alias db-prod.eus.v.co.zw, got %v", h.Aliases)
	}
	if h.HostName != "db-prod.eus.v.co.zw" {
		t.Errorf("expected hostname db-prod.eus.v.co.zw, got %s", h.HostName)
	}
	if h.User != "deploy" {
		t.Errorf("expected user deploy, got %s", h.User)
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

func writeConfigFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func aliasesOf(hosts []SSHHost) [][]string {
	var all [][]string
	for _, h := range hosts {
		all = append(all, h.Aliases)
	}
	return all
}

func TestParseConfig_SeveralNamesOnOneHostLine(t *testing.T) {
	f := writeConfigFile(t, filepath.Join(t.TempDir(), "config"),
		"Host web web-prod \"web prod\"\n    HostName 10.0.0.7\n    User deploy\n")

	hosts, err := ParseConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 SSH Host, got %d", len(hosts))
	}
	if !slices.Equal(hosts[0].Aliases, []string{"web", "web-prod", "web prod"}) {
		t.Errorf("Aliases = %q", hosts[0].Aliases)
	}
	if hosts[0].HostName != "10.0.0.7" || hosts[0].User != "deploy" {
		t.Errorf("settings not applied to SSH Host: %+v", hosts[0])
	}
}

func TestParseConfig_WildcardAndNegatedNamesAreNotAliases(t *testing.T) {
	f := writeConfigFile(t, filepath.Join(t.TempDir(), "config"), `Host db db-* !db-old db?
    User dba
Host *.internal !bastion
    User nobody
Host web
    User deploy
`)

	hosts, err := ParseConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]string{{"db"}, {"web"}}
	if got := aliasesOf(hosts); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("aliases = %q, want %q", got, want)
	}
	if hosts[0].User != "dba" || hosts[1].User != "deploy" {
		t.Errorf("settings leaked across Host blocks: %+v", hosts)
	}
}

func TestParseConfig_FollowsInclude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	writeConfigFile(t, filepath.Join(sshDir, "conf.d", "a.conf"), "Host alpha\n    User a\n")
	writeConfigFile(t, filepath.Join(sshDir, "conf.d", "b.conf"), "Host beta\n    User b\n")
	writeConfigFile(t, filepath.Join(sshDir, "extra"), "Host gamma\n")
	abs := writeConfigFile(t, filepath.Join(t.TempDir(), "abs.conf"), "Host delta\n")
	f := writeConfigFile(t, filepath.Join(sshDir, "config"),
		"include conf.d/*.conf\nInclude ~/.ssh/extra missing.conf "+abs+"\nHost local\n")

	hosts, err := ParseConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]string{{"alpha"}, {"beta"}, {"gamma"}, {"delta"}, {"local"}}
	if got := aliasesOf(hosts); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("aliases = %q, want %q", got, want)
	}
	if hosts[0].User != "a" || hosts[1].User != "b" {
		t.Errorf("included settings not applied: %+v", hosts[:2])
	}
}

func TestParseConfig_IncludeInsideHostBlockResumesThatBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	writeConfigFile(t, filepath.Join(sshDir, "inner"), "Port 2200\nHost inner\n    User i\n")
	f := writeConfigFile(t, filepath.Join(sshDir, "config"), "Host outer\n    Include inner\n    User o\n")

	hosts, err := ParseConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 SSH Hosts, got %q", aliasesOf(hosts))
	}
	if hosts[0].Port != 2200 || hosts[0].User != "o" {
		t.Errorf("outer = %+v, want port 2200 and user o", hosts[0])
	}
	if hosts[1].User != "i" {
		t.Errorf("inner = %+v, want user i", hosts[1])
	}
}

func TestParseConfig_IncludeLoopFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	f := writeConfigFile(t, filepath.Join(home, ".ssh", "config"), "Host a\nInclude config\n")

	if _, err := ParseConfig(f); err == nil || !strings.Contains(err.Error(), "Include") {
		t.Fatalf("expected an Include depth error, got %v", err)
	}
}
