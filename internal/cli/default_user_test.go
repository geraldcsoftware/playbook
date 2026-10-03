package cli

import (
	"os/user"
	"strings"
	"testing"
)

// runForInventory runs a single-play playbook against db01 and returns the
// Generated Inventory ansible-playbook received.
func runForInventory(t *testing.T, h *harness) string {
	t.Helper()
	pb := h.WritePlaybook("site.yml", "- name: Database\n  hosts: db01\n  tasks: []\n")
	if err := h.Run("run", pb, "--no-preflight"); err != nil {
		t.Fatalf("run: %v", err)
	}
	return h.Inventory()
}

func assertInventoryUser(t *testing.T, inventory, want string) {
	t.Helper()
	if !strings.Contains(inventory, " ansible_user="+want+"\n") {
		t.Errorf("Generated Inventory =\n%s\nwant ansible_user=%s", inventory, want)
	}
}

func TestRun_DefaultUser_LocalAccountWhenNotConfigured(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n")
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nport 22")
	t.Setenv("USER", "not-the-local-account")

	local, err := user.Current()
	if err != nil || local.Username == "" {
		t.Skipf("cannot look up the local account: %v", err)
	}

	assertInventoryUser(t, runForInventory(t, h), local.Username)
}

func TestRun_DefaultUser_FromConfigFile(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n")
	h.WriteConfig("default_user: deploy\n")
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nport 22")

	assertInventoryUser(t, runForInventory(t, h), "deploy")
}

func TestRun_DefaultUser_SSHHostUserWins(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n    User dbadmin\n")
	h.WriteConfig("default_user: deploy\n")
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nport 22\nuser dbadmin")

	assertInventoryUser(t, runForInventory(t, h), "dbadmin")
}
