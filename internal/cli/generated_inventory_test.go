package cli

import (
	"slices"
	"strings"
	"testing"
)

// runPlaybookHosts runs a single-play playbook against hosts, with the
// pre-flight skipped, and returns what run printed.
func runPlaybookHosts(t *testing.T, h *harness, hosts string) (string, error) {
	t.Helper()
	pb := h.WritePlaybook("site.yml", "- name: Site\n  hosts: "+hosts+"\n  tasks: []\n")
	return captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
}

func TestRun_GeneratedInventoryNamesEachSSHAlias(t *testing.T) {
	h := newHarness(t)
	h.WriteConfig("default_user: operator\ncredential_provider: aac\n")
	h.WriteSSHConfig(`Host web
    HostName 10.0.0.7
    Port 2222
    IdentityFile ~/.ssh/id_web
    User www
Host db db-primary
    HostName 10.0.0.5
`)
	h.SetSSHEffectiveConfig("web", "hostname 10.0.0.7\nport 2222\nuser www\nidentityfile ~/.ssh/id_web")
	h.SetSSHEffectiveConfig("db", "hostname 10.0.0.5\nport 22\nuser localname")

	if out, err := runPlaybookHosts(t, h, "[web, db]"); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	// Only the alias and ansible_user: OpenSSH supplies everything else.
	if got, want := h.Inventory(), "web ansible_user=www\ndb ansible_user=operator\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
	if got, want := h.SSHCalls(), []string{"-G web", "-G db"}; !slices.Equal(got, want) {
		t.Errorf("ssh calls = %q, want %q", got, want)
	}
}

func TestRun_GeneratedInventoryUserFromWildcardHostBlock(t *testing.T) {
	cases := map[string]string{
		"Host *":          "Host db01\n    HostName 10.0.0.5\nHost *\n    User everyone\n",
		"Host d?01 !db02": "Host db01\n    HostName 10.0.0.5\nHost d?01 !db02\n    User everyone\n",
		"top level":       "User everyone\nHost db01\n    HostName 10.0.0.5\n",
		"Match block":     "Host db01\n    HostName 10.0.0.5\nMatch user root\n    User everyone\n",
		"Host db*":        "Host db*\n    User everyone\nHost db01\n    HostName 10.0.0.5\n",
	}
	for name, sshConfig := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.WriteConfig("default_user: operator\n")
			h.WriteSSHConfig(sshConfig)
			h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nuser everyone")

			if out, err := runPlaybookHosts(t, h, "db01"); err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}
			if got, want := h.Inventory(), "db01 ansible_user=everyone\n"; got != want {
				t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestRun_GeneratedInventoryUsesDefaultUserWhenNoUserApplies(t *testing.T) {
	h := newHarness(t)
	h.WriteConfig("default_user: operator\n")
	// Neither block's User applies to db01: one is negated for it, the
	// other's pattern does not match it.
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\nHost * !db01\n    User everyone\nHost web-*\n    User www\n")
	// ssh -G always reports a user, here the local login name.
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nuser localname")

	if out, err := runPlaybookHosts(t, h, "db01"); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got, want := h.Inventory(), "db01 ansible_user=operator\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
}

func TestRun_FailedSSHSettingsLookupIsAHostResolutionFailure(t *testing.T) {
	h := newHarness(t)
	h.WriteConfig("default_user: operator\ncredential_provider: aac\n")
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\nHost web01\n    HostName 10.0.0.7\n")
	h.SetSSHEffectiveConfig("web01", "hostname 10.0.0.7\nuser localname")
	// No ssh -G fixture for db01, so the fake ssh exits 255 for it.

	out, err := runPlaybookHosts(t, h, "[db01, web01, mail]")
	if err == nil || !strings.Contains(err.Error(), "2 Playbook Host(s)") {
		t.Fatalf("expected a Host Resolution failure for 2 Playbook Hosts, got %v\n%s", err, out)
	}
	for _, want := range []string{
		"could not read the SSH settings for SSH Alias 'db01': ssh -G db01: fake ssh: no -G fixture for db01",
		"no SSH Alias named 'mail'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("run output missing %q:\n%s", want, out)
		}
	}
	assertNothingExecuted(t, h, out)
}
