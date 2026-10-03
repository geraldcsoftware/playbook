package inventory

import (
	"os"
	"testing"

	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

func readInventory(t *testing.T, targets []ssh.ResolvedHost) string {
	t.Helper()
	path, cleanup, err := Generate(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading inventory: %v", err)
	}
	return string(data)
}

func TestGenerate_EntriesAreSSHAliasesWithOnlyTheUser(t *testing.T) {
	targets := []ssh.ResolvedHost{
		{Alias: "db-prod", Hostname: "db-prod.eus.v.co.zw", User: "deploy", IdentityFile: "~/.ssh/id_rsa_db_prod", Port: 2222},
		{Alias: "web-01", Hostname: "web-01.eus.v.co.zw", User: "www", Port: 22},
	}

	got := readInventory(t, targets)
	want := "db-prod ansible_user=deploy\nweb-01 ansible_user=www\n"
	if got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
}

func TestGenerate_Cleanup(t *testing.T) {
	targets := []ssh.ResolvedHost{{Alias: "tmp", User: "user"}}

	path, cleanup, err := Generate(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatal("inventory file should exist before cleanup")
	}

	cleanup()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("inventory file should be deleted after cleanup")
	}
}
