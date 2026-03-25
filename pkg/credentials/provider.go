package credentials

import "os/exec"

type Provider interface {
	Wrap(itemID string, cmdName string, args []string, envMapping map[string]string) (*exec.Cmd, error)
}
