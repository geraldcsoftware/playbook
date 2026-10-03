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

// Config is the operator's SSH client configuration as the reader
// understands it: its SSH Hosts, plus enough of every block, wildcard ones
// included, to tell whether a User applies to a given SSH Alias.
type Config struct {
	// Hosts are the SSH Hosts in file order.
	Hosts []SSHHost

	blocks []block
	// userBlocks indexes, in file order, the block of every User keyword.
	userBlocks []int
}

// block is one section of the configuration: the top level, a Host line or
// a Match line.
type block struct {
	// patterns are the Host line's patterns; nil for the top level.
	patterns []string
	// match marks a Match block, whose criteria the reader does not
	// evaluate.
	match bool
}

// SetsUser reports whether the configuration applies a User to alias from
// any block that can match it: the top level, a Host block whose patterns
// match alias (wildcard blocks included) or a Match block. Match criteria
// are not evaluated, so a User in any Match block counts; `ssh -G` then
// reports what OpenSSH actually applies.
func (c Config) SetsUser(alias string) bool {
	for _, i := range c.userBlocks {
		b := c.blocks[i]
		if b.match || b.patterns == nil || hostPatternsMatch(b.patterns, alias) {
			return true
		}
	}
	return false
}

// hostPatternsMatch applies a Host line's patterns to name as OpenSSH does:
// some pattern must match and no negated pattern may.
func hostPatternsMatch(patterns []string, name string) bool {
	matched := false
	for _, p := range patterns {
		if negated, ok := strings.CutPrefix(p, "!"); ok {
			if globMatch(negated, name) {
				return false
			}
			continue
		}
		if globMatch(p, name) {
			matched = true
		}
	}
	return matched
}

// globMatch reports whether name matches pattern, where '*' matches any
// run of characters and '?' any one character, as in OpenSSH.
func globMatch(pattern, name string) bool {
	p, n := []rune(pattern), []rune(name)
	// star and resume record the last '*' seen, for backtracking.
	star, resume := -1, 0
	i, j := 0, 0
	for j < len(n) {
		switch {
		case i < len(p) && (p[i] == '?' || p[i] == n[j]):
			i++
			j++
		case i < len(p) && p[i] == '*':
			star, resume = i, j
			i++
		case star >= 0:
			resume++
			i, j = star+1, resume
		default:
			return false
		}
	}
	for i < len(p) && p[i] == '*' {
		i++
	}
	return i == len(p)
}

// ParseConfig reads the SSH client configuration at path, following Include
// directives, and returns its SSH Hosts in file order.
func ParseConfig(path string) ([]SSHHost, error) {
	c, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return c.Hosts, nil
}

// LoadConfig reads the SSH client configuration at path, following Include
// directives.
func LoadConfig(path string) (Config, error) {
	p := &configParser{current: -1, config: Config{blocks: []block{{}}}}
	if err := p.parseFile(path, 0); err != nil {
		return Config{}, err
	}
	return p.config, nil
}

type configParser struct {
	config Config
	// current indexes the SSH Host whose settings are being read, or is -1
	// when outside any SSH Host (top level or a wildcard-only Host line).
	current int
	// block indexes the block being read, whether or not it is an SSH Host.
	block int
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
			patterns := splitArgs(value)
			p.config.blocks = append(p.config.blocks, block{patterns: patterns})
			p.block = len(p.config.blocks) - 1
			aliases := literalNames(patterns)
			if len(aliases) == 0 {
				p.current = -1
				continue
			}
			p.config.Hosts = append(p.config.Hosts, SSHHost{Aliases: aliases, Port: 22})
			p.current = len(p.config.Hosts) - 1
			continue
		case "match":
			p.config.blocks = append(p.config.blocks, block{match: true})
			p.block = len(p.config.blocks) - 1
			p.current = -1
			continue
		case "include":
			if err := p.include(splitArgs(value), depth); err != nil {
				return err
			}
			continue
		}

		if strings.EqualFold(keyword, "user") {
			p.config.userBlocks = append(p.config.userBlocks, p.block)
		}

		if p.current < 0 {
			continue
		}
		current := &p.config.Hosts[p.current]

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

	enclosing, enclosingBlock := p.current, p.block
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
	p.current, p.block = enclosing, enclosingBlock
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
