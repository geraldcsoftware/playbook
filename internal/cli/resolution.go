package cli

import (
	"fmt"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
)

// resolveHosts performs Host Resolution for pb against the operator's SSH
// client configuration, returning the Targets and every failure. It is the
// one entry point run, hosts resolve and the interactive screen share; the
// error is reserved for an SSH configuration that cannot be read.
func resolveHosts(pb playbook.Playbook, cfg config.Config) ([]ssh.ResolvedHost, []ssh.HostResolutionFailure, error) {
	sshConfig, err := ssh.LoadConfig(sshConfigPath())
	if err != nil {
		return nil, nil, fmt.Errorf("parsing SSH config: %w", err)
	}
	targets, failures := ssh.ResolvePlaybook(pb, sshConfig, ssh.OpenSSHLookup{}, cfg.EffectiveDefaultUser())
	return targets, failures, nil
}
