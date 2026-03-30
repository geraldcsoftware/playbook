package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Check struct {
	Name     string
	OK       bool
	Detail   string
	Hint     string
	Optional bool
}

func CheckBinary(name string) Check {
	path, err := exec.LookPath(name)
	if err != nil {
		return Check{
			Name: name,
			OK:   false,
			Hint: fmt.Sprintf("Install %s and ensure it's on your PATH", name),
		}
	}

	version := getVersion(path, name)
	return Check{
		Name:   name,
		OK:     true,
		Detail: version,
	}
}

func CheckEnvVar(name string) Check {
	val := os.Getenv(name)
	if val == "" {
		return Check{
			Name: "$" + name,
			OK:   false,
			Hint: fmt.Sprintf("export %s=<value>", name),
		}
	}
	return Check{
		Name:   "$" + name,
		OK:     true,
		Detail: "set",
	}
}

func CheckFile(path string, label string) Check {
	info, err := os.Stat(path)
	if err != nil {
		return Check{
			Name: path,
			OK:   false,
			Hint: fmt.Sprintf("Create or check permissions for %s", path),
		}
	}
	if info.IsDir() {
		return Check{
			Name:   path,
			OK:     false,
			Detail: "is a directory, expected a file",
		}
	}
	return Check{
		Name:   path,
		OK:     true,
		Detail: label,
	}
}

// CheckProcessRunning verifies that a process with the given name is running.
func CheckProcessRunning(name string, label string) Check {
	out, err := exec.Command("pgrep", "-f", name).Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return Check{
			Name: label,
			OK:   false,
			Hint: fmt.Sprintf("Start it with: %s", name),
		}
	}
	return Check{
		Name:   label,
		OK:     true,
		Detail: "running",
	}
}

// CheckCommand runs a command and reports success/failure.
// Used to validate that a tool is properly authenticated.
func CheckCommand(name string, label string, args ...string) Check {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if len(detail) > 100 {
			detail = detail[:100] + "..."
		}
		return Check{
			Name:   label,
			OK:     false,
			Detail: detail,
			Hint:   fmt.Sprintf("Command failed: %s %s", name, strings.Join(args, " ")),
		}
	}
	return Check{
		Name:   label,
		OK:     true,
		Detail: "authenticated",
	}
}

func getVersion(path string, name string) string {
	for _, flag := range []string{"--version", "version"} {
		out, err := exec.Command(path, flag).CombinedOutput()
		if err == nil {
			line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
			return line
		}
	}
	return "found"
}
