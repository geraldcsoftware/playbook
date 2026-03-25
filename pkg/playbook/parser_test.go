package playbook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_SingleHost(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "deploy.yml")
	os.WriteFile(f, []byte("- name: Deploy App\n  hosts: db-prod\n  tasks: []\n"), 0644)

	pb, err := Parse(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pb.Name != "Deploy App" {
		t.Errorf("expected name 'Deploy App', got '%s'", pb.Name)
	}
	if len(pb.Hosts) != 1 || pb.Hosts[0] != "db-prod" {
		t.Errorf("expected hosts [db-prod], got %v", pb.Hosts)
	}
}

func TestParse_MultipleHosts(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "deploy.yml")
	os.WriteFile(f, []byte("- name: Multi Deploy\n  hosts:\n    - db-prod\n    - web-01\n  tasks: []\n"), 0644)

	pb, err := Parse(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pb.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(pb.Hosts))
	}
	if pb.Hosts[0] != "db-prod" || pb.Hosts[1] != "web-01" {
		t.Errorf("unexpected hosts: %v", pb.Hosts)
	}
}

func TestParse_UnsupportedHostPattern(t *testing.T) {
	patterns := []string{"all", "web:&staging", "*.example.com", "!db-prod"}
	for _, p := range patterns {
		dir := t.TempDir()
		f := filepath.Join(dir, "play.yml")
		os.WriteFile(f, []byte("- name: Test\n  hosts: \""+p+"\"\n  tasks: []\n"), 0644)

		_, err := Parse(f)
		if err == nil {
			t.Errorf("expected error for host pattern '%s', got nil", p)
		}
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/playbook.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
