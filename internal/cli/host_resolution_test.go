package cli

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/geraldcsoftware/playbook/internal/tui"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

// Both SSH Hosts point at a closed local port, so a pre-flight that ran in
// error would fail fast rather than wait out its timeout.
const hostResolutionSSHConfig = `Host web-prod-01
    HostName 127.0.0.1
    Port 1
Host db01 db-primary
    HostName 127.0.0.1
    Port 1
`

func TestRun_PlaybookHostMustEqualAnSSHAlias(t *testing.T) {
	h := newHarness(t)
	h.WriteConfig("credential_provider: aac\n")
	h.WriteSSHConfig(hostResolutionSSHConfig)
	pb := h.WritePlaybook("site.yml", "- hosts: web\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb) })
	if err == nil {
		t.Fatalf("expected 'web' not to resolve to web-prod-01, got success:\n%s", out)
	}
	for _, want := range []string{
		"no SSH Alias named 'web' in ~/.ssh/config",
		"did you mean web-prod-01?",
		"add a Host entry for it or pass --inventory",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("run output missing %q:\n%s", want, out)
		}
	}
	assertNothingExecuted(t, h, out)
}

func TestRun_ReportsEveryUnresolvedPlaybookHostInOnePass(t *testing.T) {
	h := newHarness(t)
	h.WriteConfig("credential_provider: aac\n")
	h.WriteSSHConfig(hostResolutionSSHConfig)
	h.SetSSHEffectiveConfig("db01", "hostname 127.0.0.1\nport 1")
	pb := h.WritePlaybook("site.yml", "- hosts: [web, db01, db1]\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb) })
	if err == nil || !strings.Contains(err.Error(), "2 Playbook Host(s)") {
		t.Fatalf("expected a Host Resolution failure for 2 Playbook Hosts, got %v\n%s", err, out)
	}
	for _, want := range []string{
		"no SSH Alias named 'web'",
		"no SSH Alias named 'db1' in ~/.ssh/config (did you mean db01?)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("run output missing %q:\n%s", want, out)
		}
	}
	assertNothingExecuted(t, h, out)
}

func TestRun_AnySSHAliasOfAnSSHHostResolves(t *testing.T) {
	h := newHarness(t)
	h.WriteConfig("credential_provider: aac\n")
	h.WriteSSHConfig(hostResolutionSSHConfig)
	h.SetSSHEffectiveConfig("db-primary", "hostname 127.0.0.1\nport 1")
	pb := h.WritePlaybook("site.yml", "- hosts: db-primary\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !h.AnsibleCalled() || len(h.AACCalls()) != 1 {
		t.Errorf("expected ansible-playbook to run with one credential fetch, aac calls %q", h.AACCalls())
	}
}

// assertNothingExecuted checks that a run stopped at Host Resolution: no
// pre-flight, no credential provider and no ansible-playbook.
func assertNothingExecuted(t *testing.T, h *harness, out string) {
	t.Helper()
	if strings.Contains(out, "SSH Pre-flight") {
		t.Errorf("the SSH pre-flight ran despite Host Resolution failures:\n%s", out)
	}
	if strings.Contains(out, "Credential Injection") {
		t.Errorf("the credential provider was resolved despite Host Resolution failures:\n%s", out)
	}
	if calls := h.AACCalls(); len(calls) != 0 {
		t.Errorf("the credential provider was asked for a secret: %q", calls)
	}
	if h.AnsibleCalled() {
		t.Error("ansible-playbook ran despite Host Resolution failures")
	}
}

func TestHostsResolve_PrintsEveryFailure(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig(hostResolutionSSHConfig)
	h.SetSSHEffectiveConfig("db-primary", "hostname 127.0.0.1\nport 1")
	pb := h.WritePlaybook("site.yml", "- name: Site\n  hosts: [web, db-primary, db1]\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("hosts", "resolve", pb) })
	if err == nil {
		t.Errorf("expected hosts resolve to fail while Playbook Hosts are unresolved")
	}
	for _, want := range []string{
		"✓ db-primary → 127.0.0.1",
		"✗ web — no SSH Alias named 'web' in ~/.ssh/config (did you mean web-prod-01?)",
		"✗ db1 — no SSH Alias named 'db1' in ~/.ssh/config (did you mean db01?)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hosts resolve output missing %q:\n%s", want, out)
		}
	}
}

func TestInteractiveScreen_DoesNotOfferRunWithHostResolutionFailures(t *testing.T) {
	pb := playbook.Playbook{Name: "Site", File: "site.yml", Hosts: []string{"web"}}
	failures := []ssh.HostResolutionFailure{{PlaybookHost: "web", Suggestions: []string{"web-prod-01"}}}
	m := tui.NewModel(pb, nil, failures, "")

	view := m.View()
	if strings.Contains(view, "Run playbook") {
		t.Errorf("expected Run not to be offered, got:\n%s", view)
	}
	if !strings.Contains(view, "did you mean web-prod-01?") {
		t.Errorf("expected the failure to be shown, got:\n%s", view)
	}

	// No item on the menu chooses Run.
	for i := 0; i < 5; i++ {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if next.(tui.Model).ChosenAction() == tui.ActionRun {
			t.Fatalf("menu item %d chose Run", i)
		}
		down, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = down.(tui.Model)
	}
}

func TestInteractiveScreen_OffersRunWhenEveryPlaybookHostResolves(t *testing.T) {
	pb := playbook.Playbook{Name: "Site", File: "site.yml", Hosts: []string{"db01"}}
	targets := []ssh.ResolvedHost{{Alias: "db01", Hostname: "10.0.0.5", Port: 22}}
	if view := tui.NewModel(pb, targets, nil, "").View(); !strings.Contains(view, "Run playbook") {
		t.Errorf("expected Run to be offered, got:\n%s", view)
	}
}
