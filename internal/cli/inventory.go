package cli

import (
	"fmt"
	"strings"

	"github.com/geraldcsoftware/playbook/pkg/playbook"
)

// explicitInventory is the Explicit Inventory supplied with the global
// --inventory flag; empty means a Generated Inventory is built instead.
var explicitInventory string

const inventoryFlagUsage = "Explicit Inventory to run against; skips Host Resolution and the SSH pre-flight check (the 'inventory' setting in ansible.cfg is always ignored)"

// parsePlaybook reads the playbook, accepting any Ansible host pattern when
// an Explicit Inventory supplies the hosts.
func parsePlaybook(path string) (playbook.Playbook, error) {
	if explicitInventory != "" {
		return playbook.ParseAnyPattern(path)
	}
	return playbook.Parse(path)
}

// checkNoInventoryArgs refuses an inventory passed to ansible-playbook other
// than through --inventory, so the inventory a run uses is never ambiguous.
func checkNoInventoryArgs(defaultArgs, extraArgs []string) error {
	if arg, ok := findInventoryArg(extraArgs); ok {
		return fmt.Errorf("'%s' found in the arguments after '--': supply the inventory with 'playbook --inventory <path>' instead", arg)
	}
	if arg, ok := findInventoryArg(defaultArgs); ok {
		return fmt.Errorf("'%s' found in ansible.default_args in the config: remove it and supply the inventory with 'playbook --inventory <path>' instead", arg)
	}
	return nil
}

// findInventoryArg returns the first argument that is ansible-playbook's
// inventory option in any of its forms: -i, -i<path>, --inventory,
// --inventory=<path>, and the older --inventory-file spellings.
func findInventoryArg(args []string) (string, bool) {
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-i"),
			arg == "--inventory", strings.HasPrefix(arg, "--inventory="),
			arg == "--inventory-file", strings.HasPrefix(arg, "--inventory-file="):
			return arg, true
		}
	}
	return "", false
}
