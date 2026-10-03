package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// startAACListener runs a stand-in process whose command line contains
// "aac listen", so doctor's listener check passes, and stops it when the
// test ends.
func startAACListener(t *testing.T, h *harness) {
	t.Helper()
	script := h.writeFile(filepath.Join(h.BinDir, "listener-stub"), "#!/bin/sh\nwhile :; do sleep 1; done\n", 0o755)
	cmd := exec.Command(script, "aac", "listen")
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting aac listen stand-in: %v", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
}

// newDoctorHarness is a harness in which every doctor check other than
// the playbook config file passes.
func newDoctorHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n")
	startAACListener(t, h)
	return h
}

func configPath(h *harness) string {
	return filepath.Join(h.Home, ".config", "playbook", "config.yaml")
}

func TestDoctor_MissingConfigWarnsWithoutFailing(t *testing.T) {
	h := newDoctorHarness(t)

	out, err := captureStdout(t, func() error { return h.Run("doctor") })
	if err != nil {
		t.Fatalf("doctor failed with only a missing config file: %v\n%s", err, out)
	}

	line := lineContaining(out, configPath(h))
	if !strings.Contains(line, "WARNING:") {
		t.Errorf("expected a warning for the missing config file, got line %q\n%s", line, out)
	}
	if !strings.Contains(out, "default_user") {
		t.Errorf("expected the warning to mention default_user\n%s", out)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected doctor to pass\n%s", out)
	}
}

func TestDoctor_UnparsableConfigWarnsNamingFileAndError(t *testing.T) {
	h := newDoctorHarness(t)
	path := h.WriteConfig("default_user: [unterminated\n")

	out, err := captureStdout(t, func() error { return h.Run("doctor") })
	if err != nil {
		t.Fatalf("doctor failed with only an unparsable config file: %v\n%s", err, out)
	}

	line := lineContaining(out, "cannot parse")
	if !strings.Contains(line, "WARNING:") || !strings.Contains(line, path) || !strings.Contains(line, "yaml:") {
		t.Errorf("expected a warning naming %s and the YAML error, got line %q\n%s", path, line, out)
	}
	if !strings.Contains(out, "All checks passed") {
		t.Errorf("expected doctor to pass\n%s", out)
	}
}

func TestDoctor_ValidConfigDoesNotWarn(t *testing.T) {
	h := newDoctorHarness(t)
	h.WriteConfig("default_user: operator\n")

	out, err := captureStdout(t, func() error { return h.Run("doctor") })
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, out)
	}
	if strings.Contains(out, "WARNING") {
		t.Errorf("expected no warning for a valid config file\n%s", out)
	}
}

func TestDoctor_DoesNotCheckKeyProvisioningTools(t *testing.T) {
	h := newDoctorHarness(t)

	out, _ := captureStdout(t, func() error { return h.Run("doctor") })
	for _, tool := range []string{"ssh-keygen", "ssh-copy-id"} {
		if strings.Contains(out, tool) {
			t.Errorf("expected doctor not to check %s\n%s", tool, out)
		}
	}
}

func TestRun_MissingOrUnparsableConfigFallsBackSilently(t *testing.T) {
	cases := map[string]func(h *harness){
		"missing":    func(h *harness) {},
		"unparsable": func(h *harness) { h.WriteConfig("default_user: [unterminated\n") },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.WriteSSHConfig("Host db01\n    HostName 10.0.0.5\n    User deploy\n")
			setup(h)
			pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

			out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
			if err != nil {
				t.Fatalf("run: %v\n%s", err, out)
			}
			// The default credential provider (aac, item ID from
			// $BW_EUS_ITEM_ID) supplied the become password.
			if got := h.BecomePass(); got != harnessPassword {
				t.Errorf("ANSIBLE_BECOME_PASS = %q, want %q", got, harnessPassword)
			}
			for _, unwanted := range []string{"WARNING", "config.yaml", "default_user"} {
				if strings.Contains(out, unwanted) {
					t.Errorf("expected run to say nothing about the config file, found %q\n%s", unwanted, out)
				}
			}
		})
	}
}

func lineContaining(out, substr string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}
