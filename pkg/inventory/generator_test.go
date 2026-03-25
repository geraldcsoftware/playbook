package inventory

import (
	"os"
	"strings"
	"testing"

	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

func TestGenerate_SingleHost(t *testing.T) {
	hosts := []ssh.ResolvedHost{
		{Alias: "db-prod", Hostname: "db-prod.eus.v.co.zw", User: "gchifanzwa", IdentityFile: "~/.ssh/id_rsa_db_prod", Port: 22},
	}

	path, cleanup, err := Generate("db-prod", hosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading inventory: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[db-prod]") {
		t.Error("expected [db-prod] group header")
	}
	if !strings.Contains(content, "db-prod.eus.v.co.zw ansible_user=gchifanzwa ansible_ssh_private_key_file=~/.ssh/id_rsa_db_prod") {
		t.Errorf("expected inline host vars, got:\n%s", content)
	}
}

func TestGenerate_MultipleHosts(t *testing.T) {
	hosts := []ssh.ResolvedHost{
		{Alias: "db-prod", Hostname: "db-prod.eus.v.co.zw", User: "gchifanzwa", IdentityFile: "~/.ssh/key1", Port: 22},
		{Alias: "web-01", Hostname: "web-01.eus.v.co.zw", User: "gchifanzwa", IdentityFile: "~/.ssh/key2", Port: 22},
	}

	path, cleanup, err := Generate("mygroup", hosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "[mygroup]") {
		t.Error("expected [mygroup] group header")
	}
	if !strings.Contains(content, "db-prod.eus.v.co.zw") {
		t.Error("expected first host")
	}
	if !strings.Contains(content, "web-01.eus.v.co.zw") {
		t.Error("expected second host")
	}
}

func TestGenerate_CustomPort(t *testing.T) {
	hosts := []ssh.ResolvedHost{
		{Alias: "custom", Hostname: "custom.example.com", User: "user", IdentityFile: "~/.ssh/key", Port: 2222},
	}

	path, cleanup, err := Generate("custom", hosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "ansible_port=2222") {
		t.Error("expected ansible_port=2222 for non-default port")
	}
}

func TestGenerate_Cleanup(t *testing.T) {
	hosts := []ssh.ResolvedHost{
		{Alias: "tmp", Hostname: "tmp.example.com", User: "user", Port: 22},
	}

	path, cleanup, err := Generate("tmp", hosts)
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
