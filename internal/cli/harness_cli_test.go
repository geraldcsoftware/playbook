package cli

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// TestRun_Characterisation_SingleHost_CurrentInventory pins down what `run`
// does today for one literal Playbook Host. The Generated Inventory it
// asserts (entry named after the SSH Host's HostName, no ssh -G lookup)
// predates ADR 0001 and is expected to change with the Host Resolution
// rework; update this test then rather than treating it as a regression.
func TestRun_Characterisation_SingleHost_CurrentInventory(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n    User deploy\n")
	h.WriteConfig("default_user: operator\ncredential_provider: aac\n")
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nport 22\nuser deploy")
	pb := h.WritePlaybook("site.yml", "- name: Database\n  hosts: db01\n  tasks: []\n")

	if err := h.Run("run", pb, "--no-preflight"); err != nil {
		t.Fatalf("run: %v", err)
	}

	args := h.AnsibleArgs()
	if len(args) != 3 || args[0] != pb || args[1] != "--inventory" {
		t.Fatalf("ansible-playbook argv = %q, want [%s --inventory <path>]", args, pb)
	}

	if got, want := h.Inventory(), "[db01]\n10.0.0.5 ansible_user=deploy\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}

	if got := h.BecomePass(); got != harnessPassword {
		t.Errorf("ANSIBLE_BECOME_PASS = %q, want %q", got, harnessPassword)
	}

	if calls := h.SSHCalls(); len(calls) != 0 {
		t.Errorf("expected no ssh calls today, got %q", calls)
	}
}

func TestHarness_AnsibleExitCodePropagates(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n    User deploy\n")
	h.SetAnsibleExitCode(4)
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	err := h.Run("run", pb, "--no-preflight")
	if err == nil || !strings.Contains(err.Error(), "exited with code 4") {
		t.Fatalf("expected ansible-playbook exit code 4 to surface, got %v", err)
	}
}

func TestHarness_ExtraArgsReachAnsible(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n    User deploy\n")
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	if err := h.Run("run", pb, "--no-preflight", "--", "--check", "-e", "x=1 y"); err != nil {
		t.Fatalf("run: %v", err)
	}

	args := h.AnsibleArgs()
	if !slices.Equal(args[3:], []string{"--check", "-e", "x=1 y"}) {
		t.Errorf("extra args = %q", args[3:])
	}
}

func TestHarness_FakeSSHServesEffectiveConfig(t *testing.T) {
	h := newHarness(t)
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nuser deploy")

	out, err := exec.Command("ssh", "-G", "db01").Output()
	if err != nil {
		t.Fatalf("ssh -G db01: %v", err)
	}
	if string(out) != "hostname 10.0.0.5\nuser deploy\n" {
		t.Errorf("ssh -G db01 printed %q", out)
	}

	if err := exec.Command("ssh", "-G", "web01").Run(); err == nil {
		t.Error("expected ssh -G for an SSH Alias without a fixture to fail")
	}

	if got, want := h.SSHCalls(), []string{"-G db01", "-G web01"}; !slices.Equal(got, want) {
		t.Errorf("ssh calls = %q, want %q", got, want)
	}
}

func TestNewRootCmd_RebindsFlagDefaults(t *testing.T) {
	t.Cleanup(resetFlags)
	noPreflight = true
	credentialProvider = "bws"

	newRootCmd()

	if noPreflight || credentialProvider != "" {
		t.Errorf("newRootCmd kept stale flag values: noPreflight=%v credentialProvider=%q", noPreflight, credentialProvider)
	}
}
