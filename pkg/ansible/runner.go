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

// BuildArgs constructs the argument list for ansible-playbook.
func BuildArgs(playbookFile string, inventoryPath string, extraArgs []string) []string {
	args := []string{
		playbookFile,
		"--inventory", inventoryPath,
	}
	args = append(args, extraArgs...)
	return args
}

// BuildCmd creates an exec.Cmd for ansible-playbook with the become password
// injected via the ANSIBLE_BECOME_PASS environment variable.
func (r *Runner) BuildCmd(password string, playbookFile string, inventoryPath string, extraArgs []string) (*exec.Cmd, error) {
	ansibleArgs := BuildArgs(playbookFile, inventoryPath, extraArgs)

	cmd := exec.Command("ansible-playbook", ansibleArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Inherit current environment and add the become password
	cmd.Env = append(os.Environ(), "ANSIBLE_BECOME_PASS="+password)

	return cmd, nil
}

// Run fetches the credential, builds the command, and executes it.
// Returns the exit code from ansible-playbook.
func (r *Runner) Run(playbookFile string, inventoryPath string, extraArgs []string) (int, error) {
	password, err := r.provider.Fetch()
	if err != nil {
		return 1, fmt.Errorf("fetching credential: %w", err)
	}

	cmd, err := r.BuildCmd(password, playbookFile, inventoryPath, extraArgs)
	if err != nil {
		return 1, fmt.Errorf("building command: %w", err)
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("running ansible-playbook: %w", err)
	}

	return 0, nil
}
