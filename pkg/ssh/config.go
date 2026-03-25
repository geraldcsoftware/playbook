package ssh

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type SSHHost struct {
	Alias        string
	HostName     string
	User         string
	IdentityFile string
	Port         int
}

func ParseConfig(path string) ([]SSHHost, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ssh config: %w", err)
	}
	defer f.Close()

	var hosts []SSHHost
	var current *SSHHost

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split into keyword and argument (supports space, tab, and = delimiters)
		var keyword, value string
		if idx := strings.IndexAny(line, " \t="); idx > 0 {
			keyword = strings.TrimSpace(line[:idx])
			value = strings.TrimSpace(line[idx+1:])
		} else {
			continue
		}

		if strings.EqualFold(keyword, "Host") {
			if current != nil {
				hosts = append(hosts, *current)
			}

			if strings.Contains(value, "*") || strings.Contains(value, "?") {
				current = nil
				continue
			}

			current = &SSHHost{
				Alias: value,
				Port:  22,
			}
			continue
		}

		if current == nil {
			continue
		}

		switch strings.ToLower(keyword) {
		case "hostname":
			current.HostName = value
		case "user":
			current.User = value
		case "identityfile":
			current.IdentityFile = value
		case "port":
			if p, err := strconv.Atoi(value); err == nil {
				current.Port = p
			}
		}
	}

	if current != nil {
		hosts = append(hosts, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ssh config: %w", err)
	}

	return hosts, nil
}
