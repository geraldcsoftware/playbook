package ssh

import (
	"path/filepath"
	"testing"
)

func TestParseEffectiveSettings(t *testing.T) {
	out := []byte("user deploy\nHostName 10.0.0.5\nidentityfile ~/.ssh/a\nidentityfile ~/.ssh/b\n\nproxycommand ssh -W %h:%p bastion\n")

	got := parseEffectiveSettings(out)
	want := map[string]string{
		"user":         "deploy",
		"hostname":     "10.0.0.5",
		"identityfile": "~/.ssh/a",
		"proxycommand": "ssh -W %h:%p bastion",
	}
	if len(got) != len(want) {
		t.Errorf("settings = %q, want %q", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("settings[%q] = %q, want %q", k, got[k], v)
		}
	}
	if got.User() != "deploy" {
		t.Errorf("User() = %q, want deploy", got.User())
	}
}

func TestEffectiveSettings_HostNameAndPort(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings EffectiveSettings
		hostname string
		port     int
	}{
		{"both set", EffectiveSettings{"hostname": "10.0.0.5", "port": "2222"}, "10.0.0.5", 2222},
		{"neither set", EffectiveSettings{}, "", 0},
		{"port not a number", EffectiveSettings{"port": "ssh"}, "", 0},
		{"port out of range", EffectiveSettings{"port": "70000"}, "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.settings.HostName(); got != tc.hostname {
				t.Errorf("HostName() = %q, want %q", got, tc.hostname)
			}
			if got := tc.settings.Port(); got != tc.port {
				t.Errorf("Port() = %d, want %d", got, tc.port)
			}
		})
	}
}

func TestConfig_SetsUser(t *testing.T) {
	f := writeConfigFile(t, filepath.Join(t.TempDir(), "config"), `Host db01
    HostName 10.0.0.5
Host web-* !web-legacy
    User www
Host cache?
    User redis
Host bastion
    User jump
`)
	c, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for alias, want := range map[string]bool{
		"db01":       false,
		"web-01":     true,
		"web-legacy": false,
		"web":        false,
		"cache1":     true,
		"cache12":    false,
		"bastion":    true,
	} {
		if got := c.SetsUser(alias); got != want {
			t.Errorf("SetsUser(%q) = %v, want %v", alias, got, want)
		}
	}
}

func TestConfig_SetsUserFromTopLevelOrIncludedBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	writeConfigFile(t, filepath.Join(sshDir, "inner"), "User o\nHost other\n")
	f := writeConfigFile(t, filepath.Join(sshDir, "config"), "Host outer\n    Include inner\nHost plain\n")

	c, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.SetsUser("outer") {
		t.Error("expected the included User to apply to the enclosing Host block")
	}
	if c.SetsUser("plain") || c.SetsUser("other") {
		t.Error("expected the included User to apply only to the enclosing Host block")
	}

	top := writeConfigFile(t, filepath.Join(t.TempDir(), "config"), "User everyone\nHost plain\n")
	if c, err := LoadConfig(top); err != nil || !c.SetsUser("plain") {
		t.Errorf("expected a top-level User to apply to every alias (err %v)", err)
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"web-*", "web-01", true},
		{"web-*", "web", false},
		{"*.internal", "db.internal", true},
		{"*.internal", "db.internal.example", false},
		{"d?01", "db01", true},
		{"d?01", "d01", false},
		{"a*b*c", "aXbYbZc", true},
		{"a*b*c", "aXbYbZ", false},
		{"db01", "db01", true},
		{"db01", "DB01", false},
	}
	for _, tt := range tests {
		if got := globMatch(tt.pattern, tt.name); got != tt.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}
