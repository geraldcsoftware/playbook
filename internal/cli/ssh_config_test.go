package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_ResolvesEitherNameOnAMultiNameHostLine(t *testing.T) {
	for _, name := range []string{"web", "web-prod"} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.WriteSSHConfig("Host web web-prod\n    HostName 10.0.0.7\n    User deploy\n")
			h.SetSSHEffectiveConfig(name, "hostname 10.0.0.7\nuser deploy")
			pb := h.WritePlaybook("site.yml", "- hosts: "+name+"\n  tasks: []\n")

			if err := h.Run("run", pb, "--no-preflight"); err != nil {
				t.Fatalf("run: %v", err)
			}
			if got, want := h.Inventory(), name+" ansible_user=deploy\n"; got != want {
				t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestRun_ResolvesSSHHostFromIncludedFile(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Include conf.d/*.conf\n")
	h.writeFile(filepath.Join(h.Home, ".ssh", "conf.d", "db.conf"), "Host db01\n    HostName 10.0.0.5\n    User dba\n", 0o600)
	h.SetSSHEffectiveConfig("db01", "hostname 10.0.0.5\nuser dba")
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	if err := h.Run("run", pb, "--no-preflight"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, want := h.Inventory(), "db01 ansible_user=dba\n"; got != want {
		t.Errorf("Generated Inventory =\n%s\nwant\n%s", got, want)
	}
}

func TestRun_WildcardNamesAreNotSSHAliases(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig("Host *\n    User everyone\nHost !bastion\n    User nobody\n")
	pb := h.WritePlaybook("site.yml", "- hosts: \"!bastion\"\n  tasks: []\n")

	if err := h.Run("run", pb, "--no-preflight"); err == nil {
		t.Fatal("expected a negated Host name not to resolve")
	}
	if h.AnsibleCalled() {
		t.Error("ansible-playbook ran although no SSH Host matched")
	}
}

func TestHostsList_GroupsAliasesBySSHHost(t *testing.T) {
	h := newHarness(t)
	h.WriteSSHConfig(`Host web web-prod web-*
    HostName 10.0.0.7
    User deploy
Host * !bastion
    User everyone
Include extra/*
`)
	h.writeFile(filepath.Join(h.Home, ".ssh", "extra", "db"), "Host db01 db\n    HostName 10.0.0.5\n    Port 2222\n", 0o600)

	out, err := captureStdout(t, func() error { return h.Run("hosts", "list") })
	if err != nil {
		t.Fatalf("hosts list: %v", err)
	}

	for _, want := range []string{
		"  web, web-prod → 10.0.0.7 (user: deploy, port: 22)\n",
		"  db01, db → 10.0.0.5 (user: , port: 2222)\n",
		"2 hosts found",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hosts list output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"*", "!bastion"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("hosts list shows %q as an SSH Alias:\n%s", unwanted, out)
		}
	}
}
