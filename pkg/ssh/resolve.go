package ssh

import (
	"fmt"
	"strings"
)

type ResolvedHost struct {
	Alias        string
	Hostname     string
	User         string
	IdentityFile string
	Port         int
}

type AmbiguousMatchError struct {
	Query      string
	Candidates []string
}

func (e *AmbiguousMatchError) Error() string {
	return fmt.Sprintf("ambiguous host '%s' matches multiple entries: %s — be more specific", e.Query, strings.Join(e.Candidates, ", "))
}

// Resolve matches a Playbook Host to an SSH Host by any of its SSH Aliases:
// an exact match wins, otherwise a unique SSH Host with an alias containing
// it. The ResolvedHost carries the SSH Alias that matched.
func Resolve(alias string, hosts []SSHHost, defaultUser string) ([]ResolvedHost, error) {
	for _, h := range hosts {
		for _, a := range h.Aliases {
			if a == alias {
				return []ResolvedHost{toResolved(a, h, defaultUser)}, nil
			}
		}
	}

	var candidates []SSHHost
	var matched []string
	for _, h := range hosts {
		for _, a := range h.Aliases {
			if strings.Contains(a, alias) {
				candidates = append(candidates, h)
				matched = append(matched, a)
				break
			}
		}
	}

	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("no SSH config entry matches '%s' — add a Host entry for it to ~/.ssh/config", alias)
	case 1:
		return []ResolvedHost{toResolved(matched[0], candidates[0], defaultUser)}, nil
	default:
		return nil, &AmbiguousMatchError{Query: alias, Candidates: matched}
	}
}

func toResolved(alias string, h SSHHost, defaultUser string) ResolvedHost {
	hostname := h.HostName
	if hostname == "" {
		hostname = alias
	}
	user := h.User
	if user == "" {
		user = defaultUser
	}
	port := h.Port
	if port == 0 {
		port = 22
	}
	return ResolvedHost{
		Alias:        alias,
		Hostname:     hostname,
		User:         user,
		IdentityFile: h.IdentityFile,
		Port:         port,
	}
}
