package cli

import (
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// preflightSSHConfig sets Port only in wildcard blocks, so a pre-flight
// that read the port from the db01 block itself would find none, and one
// that took the first wildcard Port would dial 2200.
const preflightSSHConfig = `Host db01
    HostName 10.0.0.5
Host db*
    Port 2200
Host *
    Port 2222
`

// sshListener accepts TCP connections on 127.0.0.1 and counts them, standing
// in for a reachable SSH server. It returns the port and a function giving
// the number of connections accepted once accepting has settled: dials have
// completed by the time a run returns, but the accept loop may lag them.
func sshListener(t *testing.T) (int, func() int64) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	var accepted atomic.Int64
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			conn.Close()
		}
	}()
	settled := func() int64 {
		for deadline := time.Now().Add(2 * time.Second); accepted.Load() == 0 && time.Now().Before(deadline); {
			time.Sleep(5 * time.Millisecond)
		}
		time.Sleep(50 * time.Millisecond)
		return accepted.Load()
	}
	return ln.Addr().(*net.TCPAddr).Port, settled
}

// closedPort returns a local port with nothing listening on it.
func closedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func newPreflightHarness(t *testing.T, port int) *harness {
	t.Helper()
	h := newHarness(t)
	h.WriteConfig("credential_provider: aac\n")
	h.WriteSSHConfig(preflightSSHConfig)
	h.SetSSHEffectiveConfig("db01", fmt.Sprintf("user deploy\nhostname 127.0.0.1\nport %d", port))
	return h
}

func TestRun_PreflightDialsTheEffectiveAddressAndPort(t *testing.T) {
	port, accepted := sshListener(t)
	h := newPreflightHarness(t, port)
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--timeout", "2") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if want := fmt.Sprintf("db01 (127.0.0.1:%d) — reachable", port); !strings.Contains(out, want) {
		t.Errorf("run output missing %q:\n%s", want, out)
	}
	if n := accepted(); n != 1 {
		t.Errorf("listener accepted %d connections, want 1", n)
	}
	if !h.AnsibleCalled() {
		t.Error("ansible-playbook did not run after a passing pre-flight")
	}
}

func TestRun_PreflightFailureStopsTheRun(t *testing.T) {
	port := closedPort(t)
	h := newPreflightHarness(t, port)
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--timeout", "2") })
	if err == nil {
		t.Fatalf("expected the pre-flight to fail against a closed port:\n%s", out)
	}
	for _, want := range []string{
		fmt.Sprintf("db01 (127.0.0.1:%d) — ", port),
		fmt.Sprintf("SSH port %d not reachable on 127.0.0.1", port),
		"Pre-flight failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("run output missing %q:\n%s", want, out)
		}
	}
	if h.AnsibleCalled() {
		t.Error("ansible-playbook ran despite a failed pre-flight")
	}
	if calls := h.AACCalls(); len(calls) != 0 {
		t.Errorf("the credential provider was asked for a secret: %q", calls)
	}
}

func TestRun_PreflightChecksADuplicatedTargetOnce(t *testing.T) {
	port, accepted := sshListener(t)
	h := newPreflightHarness(t, port)
	pb := h.WritePlaybook("site.yml", "- hosts: [db01, db01]\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--timeout", "2") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if n := strings.Count(out, "db01 (127.0.0.1:"); n != 1 {
		t.Errorf("pre-flight listed db01 %d times, want once:\n%s", n, out)
	}
	if n := accepted(); n != 1 {
		t.Errorf("listener accepted %d connections, want 1", n)
	}
}

func TestRun_NoPreflightSkipsTheCheck(t *testing.T) {
	// Nothing listens on the effective port, so a pre-flight that ran would
	// fail the run.
	h := newPreflightHarness(t, closedPort(t))
	pb := h.WritePlaybook("site.yml", "- hosts: db01\n  tasks: []\n")

	out, err := captureStdout(t, func() error { return h.Run("run", pb, "--no-preflight") })
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if strings.Contains(out, "SSH Pre-flight") {
		t.Errorf("the SSH pre-flight ran despite --no-preflight:\n%s", out)
	}
	if !h.AnsibleCalled() {
		t.Error("ansible-playbook did not run")
	}
}
