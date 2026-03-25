package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func EscapeHostname(hostname string) string {
	replacer := strings.NewReplacer(".", "_", "-", "_")
	return replacer.Replace(hostname)
}

func KeyPath(keyType string, hostname string) string {
	home, _ := os.UserHomeDir()
	escaped := EscapeHostname(hostname)
	return fmt.Sprintf("%s/.ssh/id_%s_%s", home, keyType, escaped)
}

func GenerateKey(keyType string, keyPath string) error {
	cmd := exec.Command("ssh-keygen", "-t", keyType, "-f", keyPath, "-N", "")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CopyKey(keyPath string, user string, hostname string, port int) error {
	pubKeyPath := keyPath + ".pub"
	cmd := exec.Command("ssh-copy-id", "-i", pubKeyPath, "-p", fmt.Sprintf("%d", port), fmt.Sprintf("%s@%s", user, hostname))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func VerifyConnection(hostname string, port int, user string) error {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-p", fmt.Sprintf("%d", port), fmt.Sprintf("%s@%s", user, hostname), "exit")
	return cmd.Run()
}

func BuildSSHConfigEntry(hostname string, user string, identityFile string, port int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nHost %s\n", hostname)
	fmt.Fprintf(&b, "    HostName %s\n", hostname)
	fmt.Fprintf(&b, "    User %s\n", user)
	fmt.Fprintf(&b, "    IdentityFile %s\n", identityFile)
	if port != 22 {
		fmt.Fprintf(&b, "    Port %d\n", port)
	}
	return b.String()
}

func AppendSSHConfigEntry(configPath string, entry string) error {
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening ssh config: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("writing ssh config entry: %w", err)
	}
	return nil
}
