package inventory

import (
	"fmt"
	"os"
	"strings"

	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

// Generate writes a Generated Inventory for targets to a temporary file and
// returns its path and a function that removes it. Each entry is the SSH
// Alias the Target was resolved to, with only its ansible_user: every other
// connection setting is left to OpenSSH, which applies the operator's SSH
// client configuration to the alias (see ADR 0001). Entries sit outside any
// group, which Ansible places in 'ungrouped' and 'all'.
func Generate(targets []ssh.ResolvedHost) (string, func(), error) {
	f, err := os.CreateTemp("", "playbook-inventory-*.ini")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp inventory: %w", err)
	}

	var b strings.Builder
	for _, t := range targets {
		fmt.Fprintf(&b, "%s ansible_user=%s\n", t.Alias, t.User)
	}

	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("writing inventory: %w", err)
	}
	f.Close()

	cleanup := func() {
		os.Remove(f.Name())
	}
	return f.Name(), cleanup, nil
}
