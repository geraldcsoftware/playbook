package credentials

import (
	"fmt"
	"os/exec"
	"sort"
)

type AACProvider struct {
	BinaryPath string // override for testing; defaults to "aac" via PATH lookup if empty
}

func NewAACProvider() *AACProvider {
	return &AACProvider{}
}

func (p *AACProvider) aacPath() (string, error) {
	if p.BinaryPath != "" {
		return p.BinaryPath, nil
	}
	return exec.LookPath("aac")
}

// Wrap builds an exec.Cmd that runs `aac run` wrapping the given command.
// envMapping maps credential field names (e.g. "password") to environment
// variable names (e.g. "ANSIBLE_BECOME_PASS"). The aac CLI expects the
// inverse order: --env ENV_VAR=field.
func (p *AACProvider) Wrap(itemID string, cmdName string, args []string, envMapping map[string]string) (*exec.Cmd, error) {
	if itemID == "" {
		return nil, fmt.Errorf("credential item ID is empty — set $BW_EUS_ITEM_ID")
	}

	aacArgs := []string{"run", "--id", itemID}

	// Sort for deterministic command construction
	keys := make([]string, 0, len(envMapping))
	for k := range envMapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, field := range keys {
		envVar := envMapping[field]
		// aac expects: --env ENV_VAR_NAME=credential_field
		aacArgs = append(aacArgs, "--env", fmt.Sprintf("%s=%s", envVar, field))
	}

	aacArgs = append(aacArgs, "--")
	aacArgs = append(aacArgs, cmdName)
	aacArgs = append(aacArgs, args...)

	aacBin, err := p.aacPath()
	if err != nil {
		return nil, fmt.Errorf("aac not found on PATH: %w", err)
	}

	cmd := exec.Command(aacBin, aacArgs...)
	return cmd, nil
}
