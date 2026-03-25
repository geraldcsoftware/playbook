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

func (p *AACProvider) Wrap(itemID string, cmdName string, args []string, envMapping map[string]string) (*exec.Cmd, error) {
	if itemID == "" {
		return nil, fmt.Errorf("credential item ID is empty — set $BW_EUS_ITEM_ID")
	}

	aacArgs := []string{"run", "--id", itemID}

	keys := make([]string, 0, len(envMapping))
	for k := range envMapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, field := range keys {
		envVar := envMapping[field]
		aacArgs = append(aacArgs, "--map", fmt.Sprintf("%s=%s", field, envVar))
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
