package cli

import (
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/geraldcsoftware/playbook/internal/tui"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
)

// captureStdout runs fn and returns what it printed to os.Stdout.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()
	runErr := fn()
	os.Stdout = orig
	w.Close()
	return <-done, runErr
}

const explicitInventoryContent = "[web]\nweb01 ansible_host=10.0.0.7\n"

func TestRun_ExplicitInventory_PassedToAnsibleWithoutHostResolution(t *testing.T) {
	for _, flag := range []string{"--inventory", "-i"} {
		t.Run(flag, func(t *testing.T) {
			h := newHarness(t)
			h.WriteConfig("credential_provider: aac\n")
			inv := h.WritePlaybook("inv.ini", explicitInventoryContent)
			// web01 is absent from the SSH config, so Host Resolution would fail.
			pb := h.WritePlaybook("site.yml", "- name: Web\n  hosts: web01\n  tasks: []\n")

			out, err := captureStdout(t, func() error { return h.Run("run", flag, inv, pb) })
			if err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}

			if got, want := h.AnsibleArgs(), []string{pb, "--inventory", inv}; !slices.Equal(got, want) {
				t.Errorf("ansible-playbook argv = %q, want %q", got, want)
			}
			if got := h.Inventory(); got != explicitInventoryContent {
				t.Errorf("inventory given to ansible-playbook =\n%s\nwant the Explicit Inventory", got)
			}
			if got := h.BecomePass(); got != harnessPassword {
				t.Errorf("ANSIBLE_BECOME_PASS = %q, want %q", got, harnessPassword)
			}
			if calls := h.SSHCalls(); len(calls) != 0 {
				t.Errorf("expected no ssh calls, got %q", calls)
			}
			if !strings.Contains(out, "Found: Web") {
				t.Errorf("expected the playbook to be parsed for its name, got:\n%s", out)
			}
			if !strings.Contains(out, "Skipped: hosts come from the Explicit Inventory") {
				t.Errorf("expected an SSH pre-flight skipped notice, got:\n%s", out)
			}
		})
	}
}

func TestRun_ExplicitInventory_AcceptsAnsiblePatterns(t *testing.T) {
	h := newHarness(t)
	inv := h.WritePlaybook("inv.ini", explicitInventoryContent)
	pb := h.WritePlaybook("site.yml", "- hosts: web:&prod\n  tasks: []\n")

	if _, err := captureStdout(t, func() error { return h.Run("run", "--inventory", inv, pb) }); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !h.AnsibleCalled() {
		t.Error("expected ansible-playbook to run")
	}
}

func TestRun_InventoryInPassthroughArgs_Refused(t *testing.T) {
	for _, extra := range [][]string{
		{"-i", "other.ini"},
		{"-iother.ini"},
		{"--inventory", "other.ini"},
		{"--check", "--inventory=other.ini"},
	} {
		for _, withFlag := range []bool{false, true} {
			name := strings.Join(extra, " ")
			if withFlag {
				name += " with --inventory"
			}
			t.Run(name, func(t *testing.T) {
				h := newHarness(t)
				h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n")
				pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

				args := []string{"run", pb, "--no-preflight"}
				if withFlag {
					args = append(args, "--inventory", h.WritePlaybook("inv.ini", explicitInventoryContent))
				}
				args = append(append(args, "--"), extra...)

				_, err := captureStdout(t, func() error { return h.Run(args...) })
				if err == nil || !strings.Contains(err.Error(), "playbook --inventory") {
					t.Fatalf("expected an error pointing to 'playbook --inventory', got %v", err)
				}
				if h.AnsibleCalled() {
					t.Error("ansible-playbook must not be called")
				}
			})
		}
	}
}

func TestRun_InventoryInDefaultArgs_Refused(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n")
	h.WriteConfig("ansible:\n  default_args: [\"--diff\", \"-i\", \"hosts.ini\"]\n")
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	_, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err == nil || !strings.Contains(err.Error(), "ansible.default_args") || !strings.Contains(err.Error(), "playbook --inventory") {
		t.Fatalf("expected an error naming ansible.default_args and 'playbook --inventory', got %v", err)
	}
	if h.AnsibleCalled() {
		t.Error("ansible-playbook must not be called")
	}
}

func TestHostsResolve_ExplicitInventory_ReportsSkip(t *testing.T) {
	h := newHarness(t)
	inv := h.WritePlaybook("inv.ini", explicitInventoryContent)
	pb := h.WritePlaybook("site.yml", "- name: Web\n  hosts: all\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("hosts", "resolve", "--inventory", inv, pb) })
	if err != nil {
		t.Fatalf("hosts resolve: %v", err)
	}
	if !strings.Contains(out, "Host Resolution skipped") || !strings.Contains(out, inv) {
		t.Errorf("expected a Host Resolution skipped notice naming %s, got:\n%s", inv, out)
	}
	if calls := h.SSHCalls(); len(calls) != 0 {
		t.Errorf("expected no ssh calls, got %q", calls)
	}
}

func TestInventoryFlag_NotesAnsibleCfgIgnored(t *testing.T) {
	t.Cleanup(resetFlags)
	flag := newRootCmd().PersistentFlags().Lookup("inventory")
	if flag == nil || flag.Shorthand != "i" {
		t.Fatalf("expected a persistent --inventory flag with shorthand -i, got %+v", flag)
	}
	if !strings.Contains(flag.Usage, "ansible.cfg") {
		t.Errorf("expected --inventory usage to note ansible.cfg is ignored, got %q", flag.Usage)
	}
}

func TestInteractiveScreen_ShowsExplicitInventory(t *testing.T) {
	pb := playbook.Playbook{Name: "Web", File: "site.yml", Hosts: []string{"all"}}
	view := tui.NewModel(pb, nil, nil, "inv.ini").View()

	if !strings.Contains(view, "inv.ini") || !strings.Contains(view, "Explicit Inventory") {
		t.Errorf("expected the screen to show the Explicit Inventory, got:\n%s", view)
	}
	if !strings.Contains(view, "Run playbook") {
		t.Errorf("expected Run to be offered, got:\n%s", view)
	}
}
