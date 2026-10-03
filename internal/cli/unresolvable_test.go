package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/geraldcsoftware/playbook/internal/tui"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

func newUnresolvableHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.WriteConfig("default_user: operator\ncredential_provider: aac\n")
	h.WriteSSHConfig(hostResolutionSSHConfig)
	for _, alias := range []string{"web-prod-01", "db01", "db-primary"} {
		h.SetSSHEffectiveConfig(alias, "hostname 127.0.0.1\nport 1")
	}
	return h
}

// runExpectingFailures runs pb and checks that Host Resolution stopped the
// run, reporting every one of want in its single report.
func runExpectingFailures(t *testing.T, h *harness, pb string, want ...string) string {
	t.Helper()
	out, err := captureStdout(t, func() error { return h.Run("run", pb) })
	if err == nil {
		t.Fatalf("expected Host Resolution to fail, got success:\n%s", out)
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("run output missing %q:\n%s", w, out)
		}
	}
	if n := strings.Count(out, "Host Resolution failed"); n != 1 {
		t.Errorf("expected one Host Resolution report, got %d:\n%s", n, out)
	}
	assertNothingExecuted(t, h, out)
	return out
}

func TestRun_HostPatternIsAHostResolutionFailure(t *testing.T) {
	patterns := []string{"all", "web*", "web?", "web:db", "web:&prod", "!db01", "~web.*", "web[0:2]", "db01,all"}
	var quoted []string
	want := []string{"no SSH Alias named 'mail'"}
	for _, p := range patterns {
		quoted = append(quoted, `"`+p+`"`)
		want = append(want, "'"+p+"' is an Ansible host pattern, which Host Resolution cannot match to an SSH Alias")
	}
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", "- hosts: ["+strings.Join(quoted, ", ")+", mail]\n  tasks: []\n")

	out := runExpectingFailures(t, h, pb, want...)
	if !strings.Contains(out, "pass --inventory to run against an Explicit Inventory") {
		t.Errorf("expected the pattern failure to point to --inventory:\n%s", out)
	}
	if calls := h.SSHCalls(); len(calls) != 0 {
		t.Errorf("expected no Host Pattern to be looked up, got ssh calls %q", calls)
	}
}

func TestRun_TemplatedPlaybookHostIsAHostResolutionFailure(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", "- hosts: \"{{ target }}\"\n  tasks: []\n- hosts: \"{{ groups['a,b'] | first }}\"\n  tasks: []\n- hosts: mail\n  tasks: []\n")

	runExpectingFailures(t, h, pb,
		"'{{ target }}' is templated, so its value is not known until the play runs",
		"'{{ groups['a,b'] | first }}' is templated",
		"pass --inventory to run against an Explicit Inventory",
		"no SSH Alias named 'mail'",
	)
}

func TestRun_ImportPlaybookIsAHostResolutionFailure(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", `- import_playbook: other.yml
- ansible.builtin.import_playbook: more.yml
- name: Web tier
  hosts: [db01, mail]
  tasks: []
`)

	runExpectingFailures(t, h, pb,
		"import_playbook other.yml — \033[31m✗\033[0m the playbook imports 'other.yml', and imported playbooks need --inventory for now",
		"the playbook imports 'more.yml'",
		"no SSH Alias named 'mail'",
	)
}

func TestRun_OnlyImportsIsAHostResolutionFailure(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", "- import_playbook: other.yml\n")

	runExpectingFailures(t, h, pb, "the playbook imports 'other.yml', and imported playbooks need --inventory for now")
}

func TestRun_SameSSHHostTwiceInAPlayIsAHostResolutionFailure(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", `- name: Database tier
  hosts: [db01, web-prod-01, db-primary]
  tasks: []
- hosts: db-primary, db01, mail
  tasks: []
`)

	runExpectingFailures(t, h, pb,
		"'db01' and 'db-primary' in play 'Database tier' are SSH Aliases of the same SSH Host — list it once in the play, or pass --inventory",
		"'db-primary' and 'db01' in play 2 are SSH Aliases of the same SSH Host",
		"no SSH Alias named 'mail'",
	)
}

func TestRun_SamePlaybookHostTwiceInAPlayIsNotAFailure(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", "- hosts: [db01, db01]\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got, want := h.Inventory(), "db01 ansible_user=operator\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
}

func TestRun_SameSSHHostByDifferentAliasesInDifferentPlaysRuns(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n- hosts: db-primary\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got, want := h.Inventory(), "db01 ansible_user=operator\ndb-primary ansible_user=operator\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
}

// unresolvablePlaybook holds one of every kind of failure Host Resolution
// reports, besides an unreadable SSH configuration.
const unresolvablePlaybook = `- import_playbook: other.yml
- name: Everything
  hosts: [all, "{{ target }}", db01, db-primary, mail]
  tasks: []
`

func TestRun_EveryKindOfFailureIsInOneReport(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", unresolvablePlaybook)

	out := runExpectingFailures(t, h, pb,
		"the playbook imports 'other.yml'",
		"'all' is an Ansible host pattern",
		"'{{ target }}' is templated",
		"no SSH Alias named 'mail'",
		"'db01' and 'db-primary' in play 'Everything' are SSH Aliases of the same SSH Host",
	)
	if strings.Index(out, "imports 'other.yml'") > strings.Index(out, "no SSH Alias named 'mail'") {
		t.Errorf("expected the import to be reported first:\n%s", out)
	}
}

func TestRun_ExplicitInventory_SkipsEveryHostResolutionCheck(t *testing.T) {
	h := newUnresolvableHarness(t)
	inv := h.WritePlaybook("inv.ini", explicitInventoryContent)
	pb := h.WritePlaybook("site.yml", unresolvablePlaybook)

	out, err := captureStdout(t, func() error { return h.Run("run", "--inventory", inv, pb) })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if got, want := h.AnsibleArgs(), []string{pb, "--inventory", inv}; !slices.Equal(got, want) {
		t.Errorf("ansible-playbook argv = %q, want %q", got, want)
	}
	if calls := h.SSHCalls(); len(calls) != 0 {
		t.Errorf("expected no ssh calls, got %q", calls)
	}
}

func TestRun_ExplicitInventory_RunsAPlaybookOfOnlyImports(t *testing.T) {
	h := newUnresolvableHarness(t)
	inv := h.WritePlaybook("inv.ini", explicitInventoryContent)
	pb := h.WritePlaybook("site.yml", "- import_playbook: other.yml\n")

	if out, err := captureStdout(t, func() error { return h.Run("run", "--inventory", inv, pb) }); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !h.AnsibleCalled() {
		t.Error("expected ansible-playbook to run")
	}
}

func TestHostsResolve_PrintsEveryKindOfFailure(t *testing.T) {
	h := newUnresolvableHarness(t)
	pb := h.WritePlaybook("site.yml", unresolvablePlaybook)

	out, err := captureStdout(t, func() error { return h.Run("hosts", "resolve", pb) })
	if err == nil {
		t.Errorf("expected hosts resolve to fail:\n%s", out)
	}
	for _, want := range []string{
		"✗ import_playbook other.yml — the playbook imports 'other.yml'",
		"✗ all — 'all' is an Ansible host pattern",
		"✗ {{ target }} — '{{ target }}' is templated",
		"✗ mail — no SSH Alias named 'mail'",
		"✗ db01, db-primary — 'db01' and 'db-primary' in play 'Everything'",
		"✓ db01 → 127.0.0.1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hosts resolve output missing %q:\n%s", want, out)
		}
	}
}

func TestInteractiveScreen_ShowsEveryKindOfFailure(t *testing.T) {
	pb := playbook.Playbook{Name: "Site", File: "site.yml", Imports: []string{"other.yml"}}
	failures := []ssh.HostResolutionFailure{
		{Kind: ssh.ImportedPlaybook, Import: "other.yml"},
		{Kind: ssh.HostPattern, PlaybookHost: "all"},
		{Kind: ssh.SSHHostRepeatedInPlay, PlaybookHost: "db-primary", SameSSHHostAs: "db01", PlayNumber: 1},
	}
	view := tui.NewModel(pb, nil, failures, "").View()

	for _, want := range []string{
		"import_playbook other.yml: the playbook imports 'other.yml'",
		"all: 'all' is an Ansible host pattern",
		"db01, db-primary: 'db01' and 'db-primary' in play 1",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("screen missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Run playbook") {
		t.Errorf("expected Run not to be offered:\n%s", view)
	}
}
