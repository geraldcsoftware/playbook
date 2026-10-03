package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// harnessPassword is the become password the fake aac hands out.
const harnessPassword = "harness-become-pass"

// harnessItemID is the AAC item ID exported for every harness run.
const harnessItemID = "harness-item-id"

// harness is an isolated operator environment for driving real commands
// through the cobra root command. It owns a temporary HOME and a bin
// directory, prepended to PATH, holding fake ssh, ansible-playbook and aac
// executables that record what they were called with.
type harness struct {
	t *testing.T

	Home   string // temporary HOME
	BinDir string // directory of fake executables, first on PATH
	LogDir string // where the fakes record their invocations

	sshFixtureDir string
}

// newHarness builds a fresh environment for t. HOME, PATH and the AAC item
// ID env var are set with t.Setenv, and package-level flag variables are
// reset when the test ends, so nothing leaks between tests.
func newHarness(t *testing.T) *harness {
	t.Helper()

	root := t.TempDir()
	h := &harness{
		t:             t,
		Home:          filepath.Join(root, "home"),
		BinDir:        filepath.Join(root, "bin"),
		LogDir:        filepath.Join(root, "log"),
		sshFixtureDir: filepath.Join(root, "ssh-G"),
	}
	for _, dir := range []string{h.Home, h.BinDir, h.LogDir, h.sshFixtureDir, filepath.Join(h.Home, ".ssh")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}

	t.Setenv("HOME", h.Home)
	t.Setenv("PATH", h.BinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("BW_EUS_ITEM_ID", harnessItemID)
	t.Setenv("ANSIBLE_BECOME_PASS", "")

	h.installFakes()
	t.Cleanup(resetFlags)

	return h
}

// resetFlags restores every package-level flag variable to its zero value.
// newRootCmd rebinds them to their defaults, but tests that build a
// subcommand directly would otherwise inherit a previous test's values.
func resetFlags() {
	cfgFile = ""
	verbose = false
	noPreflight = false
	credentialProvider = ""
	secretName = ""
	accessToken = ""
}

// WriteSSHConfig writes ~/.ssh/config and returns its path.
func (h *harness) WriteSSHConfig(content string) string {
	h.t.Helper()
	return h.writeFile(filepath.Join(h.Home, ".ssh", "config"), content, 0o600)
}

// WriteConfig writes ~/.config/playbook/config.yaml and returns its path.
func (h *harness) WriteConfig(content string) string {
	h.t.Helper()
	return h.writeFile(filepath.Join(h.Home, ".config", "playbook", "config.yaml"), content, 0o644)
}

// WritePlaybook writes a playbook file named name under HOME and returns
// its path.
func (h *harness) WritePlaybook(name, content string) string {
	h.t.Helper()
	return h.writeFile(filepath.Join(h.Home, name), content, 0o644)
}

// SetSSHEffectiveConfig sets what the fake `ssh -G <alias>` prints for an
// SSH Alias: OpenSSH's "keyword value" lines, e.g. "hostname 10.0.0.5".
// `ssh -G` for an alias without a fixture exits 255.
func (h *harness) SetSSHEffectiveConfig(alias, output string) {
	h.t.Helper()
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	h.writeFile(filepath.Join(h.sshFixtureDir, alias), output, 0o644)
}

// SetAnsibleExitCode makes the fake ansible-playbook exit with code.
func (h *harness) SetAnsibleExitCode(code int) {
	h.t.Helper()
	h.writeFile(filepath.Join(h.LogDir, "ansible-exit-code"), fmt.Sprintf("%d\n", code), 0o644)
}

// Run executes the playbook root command with args, as main would.
func (h *harness) Run(args ...string) error {
	h.t.Helper()
	root := newRootCmd()
	root.SetArgs(args)
	return root.Execute()
}

// AnsibleCalled reports whether the fake ansible-playbook was invoked.
func (h *harness) AnsibleCalled() bool {
	_, err := os.Stat(filepath.Join(h.LogDir, "ansible-argv"))
	return err == nil
}

// AnsibleArgs returns the argv (excluding argv[0]) of the last
// ansible-playbook invocation.
func (h *harness) AnsibleArgs() []string {
	h.t.Helper()
	return h.readLines(filepath.Join(h.LogDir, "ansible-argv"))
}

// Inventory returns a copy of the inventory file ansible-playbook was given
// with --inventory or -i, taken before playbook cleaned it up.
func (h *harness) Inventory() string {
	h.t.Helper()
	return h.readFile(filepath.Join(h.LogDir, "ansible-inventory"))
}

// BecomePass returns the ANSIBLE_BECOME_PASS ansible-playbook saw.
func (h *harness) BecomePass() string {
	h.t.Helper()
	return h.readFile(filepath.Join(h.LogDir, "ansible-become-pass"))
}

// SSHCalls returns one entry per ssh invocation, each its space-joined argv
// (excluding argv[0]). It is empty when ssh was never called.
func (h *harness) SSHCalls() []string {
	h.t.Helper()
	path := filepath.Join(h.LogDir, "ssh-calls")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return h.readLines(path)
}

func (h *harness) installFakes() {
	h.t.Helper()

	ssh := fmt.Sprintf(`#!/bin/sh
echo "$*" >> '%[1]s/ssh-calls'
if [ "$1" = "-G" ] && [ $# -eq 2 ]; then
	if [ -f '%[2]s'/"$2" ]; then
		cat '%[2]s'/"$2"
		exit 0
	fi
	echo "fake ssh: no -G fixture for $2" >&2
	exit 255
fi
echo "fake ssh: unsupported invocation: $*" >&2
exit 255
`, h.LogDir, h.sshFixtureDir)

	ansiblePlaybook := fmt.Sprintf(`#!/bin/sh
: > '%[1]s/ansible-argv'
for arg in "$@"; do
	printf '%%s\n' "$arg" >> '%[1]s/ansible-argv'
done
printf '%%s' "$ANSIBLE_BECOME_PASS" > '%[1]s/ansible-become-pass'
while [ $# -gt 0 ]; do
	case "$1" in
	--inventory|-i)
		cp "$2" '%[1]s/ansible-inventory'
		shift
		;;
	esac
	shift
done
if [ -f '%[1]s/ansible-exit-code' ]; then
	exit "$(cat '%[1]s/ansible-exit-code')"
fi
exit 0
`, h.LogDir)

	aac := fmt.Sprintf(`#!/bin/sh
echo "$*" >> '%[1]s/aac-calls'
if [ "$1" = "connect" ] && [ "$2" = "--id" ] && [ "$3" = '%[2]s' ]; then
	echo '{"credential":{"password":"%[3]s","username":"harness"},"success":true}'
	exit 0
fi
echo "fake aac: unexpected invocation: $*" >&2
exit 1
`, h.LogDir, harnessItemID, harnessPassword)

	h.writeFile(filepath.Join(h.BinDir, "ssh"), ssh, 0o755)
	h.writeFile(filepath.Join(h.BinDir, "ansible-playbook"), ansiblePlaybook, 0o755)
	h.writeFile(filepath.Join(h.BinDir, "aac"), aac, 0o755)
}

func (h *harness) writeFile(path, content string, mode os.FileMode) string {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		h.t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		h.t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

func (h *harness) readFile(path string) string {
	h.t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func (h *harness) readLines(path string) []string {
	h.t.Helper()
	content := strings.TrimSuffix(h.readFile(path), "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
