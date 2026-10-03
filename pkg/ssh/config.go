package ssh

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SSHHost is one Host entry of the operator's SSH client configuration,
// together with the connection settings that apply to it. Aliases holds its
// SSH Aliases: every literal name on its Host line, in order.
type SSHHost struct {
	Aliases      []string
	HostName     string
	User         string
	IdentityFile string
	Port         int
}

// maxIncludeDepth bounds nested Include directives, matching OpenSSH's own
// limit, so an Include loop fails rather than recursing forever.
const maxIncludeDepth = 16

// ParseConfig reads the SSH client configuration at path, following Include
// directives, and returns its SSH Hosts in file order.
func ParseConfig(path string) ([]SSHHost, error) {
	p := &configParser{current: -1}
	if err := p.parseFile(path, 0); err != nil {
		return nil, err
	}
	return p.hosts, nil
}

type configParser struct {
	hosts []SSHHost
	// current indexes the SSH Host whose settings are being read, or is -1
	// when outside any SSH Host (top level or a wildcard-only Host line).
	current int
}

func (p *configParser) parseFile(path string, depth int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening ssh config: %w", err)
	}
	defer f.Close()

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
			value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line[idx+1:]), "="))
		} else {
			continue
		}

		switch strings.ToLower(keyword) {
		case "host":
			aliases := literalNames(splitArgs(value))
			if len(aliases) == 0 {
				p.current = -1
				continue
			}
			p.hosts = append(p.hosts, SSHHost{Aliases: aliases, Port: 22})
			p.current = len(p.hosts) - 1
			continue
		case "include":
			if err := p.include(splitArgs(value), depth); err != nil {
				return err
			}
			continue
		}

		if p.current < 0 {
			continue
		}
		current := &p.hosts[p.current]

		switch strings.ToLower(keyword) {
		case "hostname":
			current.HostName = value
		case "user":
			current.User = value
		case "identityfile":
			current.IdentityFile = value
		case "port":
			if port, err := strconv.Atoi(value); err == nil {
				current.Port = port
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading ssh config: %w", err)
	}
	return nil
}

// include reads each file an Include directive names. As in OpenSSH,
// relative paths are taken from ~/.ssh, globs are expanded, files that do
// not exist are skipped, and the enclosing Host block resumes afterwards.
func (p *configParser) include(patterns []string, depth int) error {
	if depth+1 > maxIncludeDepth {
		return fmt.Errorf("reading ssh config: too many nested Include directives (limit %d)", maxIncludeDepth)
	}

	enclosing := p.current
	for _, pattern := range patterns {
		matches, err := filepath.Glob(includePath(pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if info, err := os.Stat(match); err != nil || info.IsDir() {
				continue
			}
			if err := p.parseFile(match, depth+1); err != nil {
				return err
			}
		}
	}
	p.current = enclosing
	return nil
}

func includePath(pattern string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return pattern
	}
	if pattern == "~" || strings.HasPrefix(pattern, "~/") {
		return filepath.Join(home, pattern[1:])
	}
	if filepath.IsAbs(pattern) {
		return pattern
	}
	return filepath.Join(home, ".ssh", pattern)
}

// literalNames keeps the names that can be SSH Aliases, dropping patterns
// containing wildcards and negated names.
func literalNames(names []string) []string {
	var literal []string
	for _, n := range names {
		if strings.ContainsAny(n, "*?!") {
			continue
		}
		literal = append(literal, n)
	}
	return literal
}

// splitArgs splits a directive's arguments on whitespace, treating a
// double-quoted run as one argument.
func splitArgs(value string) []string {
	var args []string
	var b strings.Builder
	inQuotes, inArg := false, false
	for _, r := range value {
		switch {
		case r == '"':
			inQuotes = !inQuotes
			inArg = true
		case (r == ' ' || r == '\t') && !inQuotes:
			if inArg {
				args = append(args, b.String())
				b.Reset()
				inArg = false
			}
		default:
			b.WriteRune(r)
			inArg = true
		}
	}
	if inArg {
		args = append(args, b.String())
	}
	return args
}
