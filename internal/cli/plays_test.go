package cli

import (
	"slices"
	"strings"
	"testing"
)

const playsSSHConfig = `Host web
    HostName 10.0.0.7
Host db
    HostName 10.0.0.5
`

func newPlaysHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.WriteConfig("default_user: operator\ncredential_provider: aac\n")
	h.WriteSSHConfig(playsSSHConfig)
	h.SetSSHEffectiveConfig("web", "hostname 10.0.0.7\nport 22")
	h.SetSSHEffectiveConfig("db", "hostname 10.0.0.5\nport 22")
	return h
}

func TestRun_ResolvesThePlaybookHostsOfEveryPlay(t *testing.T) {
	h := newPlaysHarness(t)
	pb := h.WritePlaybook("site.yml", `- name: Web tier
  hosts: web
  tasks: []
- name: Database tier
  hosts: db
  tasks: []
`)

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got, want := h.Inventory(), "web ansible_user=operator\ndb ansible_user=operator\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
	if !strings.Contains(out, "db → 10.0.0.5") {
		t.Errorf("expected the second play's Playbook Host in the resolution output:\n%s", out)
	}
	if !strings.Contains(out, "Found: Web tier") {
		t.Errorf("expected the first play's name to name the playbook:\n%s", out)
	}
}

func TestHostsResolve_ResolvesThePlaybookHostsOfEveryPlay(t *testing.T) {
	h := newPlaysHarness(t)
	pb := h.WritePlaybook("site.yml", "- hosts: web\n  tasks: []\n- hosts: db\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("hosts", "resolve", pb) })
	if err != nil {
		t.Fatalf("hosts resolve: %v\n%s", err, out)
	}
	for _, want := range []string{"✓ web → 10.0.0.7", "✓ db → 10.0.0.5"} {
		if !strings.Contains(out, want) {
			t.Errorf("hosts resolve output missing %q:\n%s", want, out)
		}
	}
}

func TestRun_CommaSeparatedPlaybookHostsResolveSeparately(t *testing.T) {
	for name, hosts := range map[string]string{
		"plain":       "web,db",
		"spaced":      "\"web , db\"",
		"empty parts": "\",web,,db,\"",
		"in a list":   "[\"web,db\"]",
	} {
		t.Run(name, func(t *testing.T) {
			h := newPlaysHarness(t)
			if out, err := runPlaybookHosts(t, h, hosts); err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}
			if got, want := h.Inventory(), "web ansible_user=operator\ndb ansible_user=operator\n"; got != want {
				t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestRun_TargetNamedByTwoPlaysAppearsOnce(t *testing.T) {
	h := newPlaysHarness(t)
	pb := h.WritePlaybook("site.yml", `- hosts: web, db
  tasks: []
- hosts: [db]
  tasks: []
`)

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got, want := h.Inventory(), "web ansible_user=operator\ndb ansible_user=operator\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
	if n := strings.Count(out, "db → 10.0.0.5"); n != 1 {
		t.Errorf("expected db once in the resolution output, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "2 host(s) resolved") {
		t.Errorf("expected 2 Targets:\n%s", out)
	}
	if got, want := h.SSHCalls(), []string{"-G web", "-G db"}; !slices.Equal(got, want) {
		t.Errorf("ssh calls = %q, want %q", got, want)
	}
}

func TestRun_UnresolvedPlaybookHostInSeveralPlaysIsReportedOnce(t *testing.T) {
	h := newPlaysHarness(t)
	pb := h.WritePlaybook("site.yml", "- hosts: mail\n  tasks: []\n- hosts: [web, mail]\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err == nil || !strings.Contains(err.Error(), "1 Playbook Host(s)") {
		t.Fatalf("expected a Host Resolution failure for 1 Playbook Host, got %v\n%s", err, out)
	}
	if n := strings.Count(out, "no SSH Alias named 'mail'"); n != 1 {
		t.Errorf("expected the failure for 'mail' once, got %d:\n%s", n, out)
	}
	assertNothingExecuted(t, h, out)
}

func TestRun_ImportPlaybookEntriesRunWithAnExplicitInventory(t *testing.T) {
	h := newPlaysHarness(t)
	inv := h.WritePlaybook("inv.ini", explicitInventoryContent)
	pb := h.WritePlaybook("site.yml", `- import_playbook: other.yml
- ansible.builtin.import_playbook: more.yml
- name: Web tier
  hosts: web
  tasks: []
`)

	out, err := captureStdout(t, func() error { return h.Run("run", "--inventory", inv, pb) })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got := h.Inventory(); got != explicitInventoryContent {
		t.Errorf("inventory given to ansible-playbook =\n%s\nwant the Explicit Inventory", got)
	}
}
