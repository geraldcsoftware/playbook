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

func Resolve(alias string, hosts []SSHHost, defaultUser string) ([]ResolvedHost, error) {
	for _, h := range hosts {
		if h.Alias == alias {
			return []ResolvedHost{toResolved(alias, h, defaultUser)}, nil
		}
	}

	var candidates []SSHHost
	for _, h := range hosts {
		if strings.Contains(h.Alias, alias) {
			candidates = append(candidates, h)
		}
	}

	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("no SSH config entry matches '%s' — run 'playbook hosts add' to add it", alias)
	case 1:
		return []ResolvedHost{toResolved(alias, candidates[0], defaultUser)}, nil
	default:
		names := make([]string, len(candidates))
		for i, c := range candidates {
			names[i] = c.Alias
		}
		return nil, &AmbiguousMatchError{Query: alias, Candidates: names}
	}
}

func toResolved(alias string, h SSHHost, defaultUser string) ResolvedHost {
	hostname := h.HostName
	if hostname == "" {
		hostname = h.Alias
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
