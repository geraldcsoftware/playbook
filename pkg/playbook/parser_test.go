package playbook

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writePlaybook(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "site.yml")
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParse_SingleHost(t *testing.T) {
	pb, err := Parse(writePlaybook(t, "- name: Deploy App\n  hosts: db-prod\n  tasks: []\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pb.Name != "Deploy App" {
		t.Errorf("expected name 'Deploy App', got '%s'", pb.Name)
	}
	if got := pb.Hosts(); !slices.Equal(got, []string{"db-prod"}) {
		t.Errorf("expected hosts [db-prod], got %v", got)
	}
}

func TestParse_MultipleHosts(t *testing.T) {
	pb, err := Parse(writePlaybook(t, "- name: Multi Deploy\n  hosts:\n    - db-prod\n    - web-01\n  tasks: []\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pb.Hosts(); !slices.Equal(got, []string{"db-prod", "web-01"}) {
		t.Errorf("unexpected hosts: %v", got)
	}
}

func TestParse_EveryPlay(t *testing.T) {
	f := writePlaybook(t, `- name: Web tier
  hosts: web
  tasks: []
- import_playbook: other.yml
- ansible.builtin.import_playbook: more.yml
- name: Database tier
  hosts: [db, web]
  tasks: []
`)

	pb, err := Parse(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []Play{
		{Name: "Web tier", Hosts: []string{"web"}},
		{Name: "Database tier", Hosts: []string{"db", "web"}},
	}
	if !slices.EqualFunc(pb.Plays, want, func(a, b Play) bool {
		return a.Name == b.Name && slices.Equal(a.Hosts, b.Hosts)
	}) {
		t.Errorf("plays = %+v, want %+v", pb.Plays, want)
	}
	if pb.Name != "Web tier" {
		t.Errorf("expected the first play's name, got %q", pb.Name)
	}
	if got, want := pb.Imports, []string{"other.yml", "more.yml"}; !slices.Equal(got, want) {
		t.Errorf("imports = %q, want %q", got, want)
	}
	if got, want := pb.Hosts(), []string{"web", "db"}; !slices.Equal(got, want) {
		t.Errorf("Hosts() = %q, want %q", got, want)
	}
}

func TestParse_CommaSeparatedHosts(t *testing.T) {
	tests := map[string][]string{
		"web,db":             {"web", "db"},
		`" web , db "`:       {"web", "db"},
		`",web,,db,"`:        {"web", "db"},
		`["web,db", mail]`:   {"web", "db", "mail"},
		"[db-prod, web-01]":  {"db-prod", "web-01"},
		"single-host-no-sep": {"single-host-no-sep"},
	}
	for hosts, want := range tests {
		pb, err := Parse(writePlaybook(t, "- hosts: "+hosts+"\n  tasks: []\n"))
		if err != nil {
			t.Fatalf("hosts %s: unexpected error: %v", hosts, err)
		}
		if got := pb.Plays[0].Hosts; !slices.Equal(got, want) {
			t.Errorf("hosts %s = %q, want %q", hosts, got, want)
		}
	}
}

func TestParse_UnsupportedHostPattern(t *testing.T) {
	patterns := []string{"all", "web:&staging", "*.example.com", "!db-prod", "web,all"}
	for _, p := range patterns {
		if _, err := Parse(writePlaybook(t, "- name: Test\n  hosts: \""+p+"\"\n  tasks: []\n")); err == nil {
			t.Errorf("expected error for host pattern '%s', got nil", p)
		}
	}
}

func TestParse_UnsupportedHostPatternInALaterPlay(t *testing.T) {
	if _, err := Parse(writePlaybook(t, "- hosts: web\n- hosts: \"db:&staging\"\n")); err == nil {
		t.Error("expected a pattern in a later play to be rejected")
	}
}

func TestParse_NoPlays(t *testing.T) {
	if _, err := Parse(writePlaybook(t, "- import_playbook: other.yml\n")); err == nil {
		t.Error("expected a playbook with no plays to be rejected")
	}
}

func TestParseAnyPattern_EveryPlay(t *testing.T) {
	pb, err := ParseAnyPattern(writePlaybook(t, "- hosts: all\n- hosts: \"web:&staging\"\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := pb.Hosts(), []string{"all", "web:&staging"}; !slices.Equal(got, want) {
		t.Errorf("Hosts() = %q, want %q", got, want)
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/playbook.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
