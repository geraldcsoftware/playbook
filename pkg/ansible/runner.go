package ansible

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/geraldcsoftware/playbook/pkg/credentials"
)

type Runner struct {
	provider credentials.Provider
}

func NewRunner(provider credentials.Provider) *Runner {
	return &Runner{provider: provider}
}

func BuildArgs(playbookFile string, inventoryPath string, extraArgs []string) []string {
	args := []string{
		playbookFile,
		"--inventory", inventoryPath,
		"--extra-vars", "ansible_become_pass={{ lookup('env', 'ANSIBLE_BECOME_PASS') }}",
	}
	args = append(args, extraArgs...)
	return args
}

func (r *Runner) Run(itemID string, playbookFile string, inventoryPath string, extraArgs []string) (int, error) {
	ansibleArgs := BuildArgs(playbookFile, inventoryPath, extraArgs)

	envMapping := map[string]string{
		"password": "ANSIBLE_BECOME_PASS",
	}

	cmd, err := r.provider.Wrap(itemID, "ansible-playbook", ansibleArgs, envMapping)
	if err != nil {
		return 1, fmt.Errorf("building command: %w", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("running ansible-playbook: %w", err)
	}

	return 0, nil
}
