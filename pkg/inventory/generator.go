package inventory

import (
	"fmt"
	"os"
	"strings"

	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

func Generate(groupName string, hosts []ssh.ResolvedHost) (string, func(), error) {
	f, err := os.CreateTemp("", "playbook-inventory-*.ini")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp inventory: %w", err)
	}

	var b strings.Builder

	fmt.Fprintf(&b, "[%s]\n", groupName)
	for _, h := range hosts {
		line := h.Hostname
		line += fmt.Sprintf(" ansible_user=%s", h.User)
		if h.IdentityFile != "" {
			line += fmt.Sprintf(" ansible_ssh_private_key_file=%s", h.IdentityFile)
		}
		if h.Port != 22 {
			line += fmt.Sprintf(" ansible_port=%d", h.Port)
		}
		fmt.Fprintln(&b, line)
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
