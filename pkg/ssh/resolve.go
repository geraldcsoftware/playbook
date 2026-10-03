package ssh

import (
	"fmt"
	"slices"
	"strings"
)

type ResolvedHost struct {
	Alias        string
	Hostname     string
	User         string
	IdentityFile string
	Port         int
}

// maxSuggestionDistance is the largest edit distance at which an SSH Alias
// is still suggested for a Playbook Host it does not contain. A short
// Playbook Host is held to fewer edits than half its length, so that
// unrelated short names are not offered.
const maxSuggestionDistance = 2

// maxSuggestions caps how many SSH Aliases a failure suggests.
const maxSuggestions = 3

// HostResolutionFailure is a Playbook Host that Host Resolution could not
// turn into a Target. Either it matched no SSH Alias, and Suggestions lists
// the nearest SSH Aliases, nearest first; or it matched one whose effective
// settings could not be read, and Err says why.
type HostResolutionFailure struct {
	PlaybookHost string
	Suggestions  []string
	Err          error
}

func (f HostResolutionFailure) Error() string {
	if f.Err != nil {
		return fmt.Sprintf("could not read the SSH settings for SSH Alias '%s': %v — fix its SSH configuration or pass --inventory", f.PlaybookHost, f.Err)
	}
	msg := fmt.Sprintf("no SSH Alias named '%s' in ~/.ssh/config", f.PlaybookHost)
	if len(f.Suggestions) > 0 {
		msg += fmt.Sprintf(" (did you mean %s?)", strings.Join(f.Suggestions, ", "))
	}
	return msg + " — add a Host entry for it or pass --inventory"
}

// Resolve performs Host Resolution: each Playbook Host resolves only to an
// SSH Alias equal to it, any alias of an SSH Host counting. Each Target's
// User is the one OpenSSH effectively applies, as lookup reports it, when
// the configuration sets a User for that alias in any matching block;
// otherwise it is defaultUser. It returns a Target for every Playbook Host
// that resolved and a failure for every one that did not, so all failures
// can be reported together; a run must not proceed while any failure
// exists. A Playbook Host listed more than once yields one Target, or one
// failure, and is looked up once: under exact matching it names a single
// SSH Alias.
func Resolve(playbookHosts []string, config Config, lookup EffectiveSettingsLookup, defaultUser string) ([]ResolvedHost, []HostResolutionFailure) {
	var targets []ResolvedHost
	var failures []HostResolutionFailure
	seen := map[string]bool{}
	for _, ph := range playbookHosts {
		if seen[ph] {
			continue
		}
		seen[ph] = true
		h, ok := matchAlias(ph, config.Hosts)
		if !ok {
			failures = append(failures, HostResolutionFailure{
				PlaybookHost: ph,
				Suggestions:  suggestAliases(ph, config.Hosts),
			})
			continue
		}
		settings, err := lookup.EffectiveSettings(ph)
		if err != nil {
			failures = append(failures, HostResolutionFailure{PlaybookHost: ph, Err: err})
			continue
		}
		user := defaultUser
		if config.SetsUser(ph) && settings.User() != "" {
			user = settings.User()
		}
		targets = append(targets, toResolved(ph, h, user))
	}
	return targets, failures
}

// matchAlias finds the first SSH Host, in file order, with an SSH Alias
// equal to playbookHost, as OpenSSH itself would.
func matchAlias(playbookHost string, hosts []SSHHost) (SSHHost, bool) {
	for _, h := range hosts {
		if slices.Contains(h.Aliases, playbookHost) {
			return h, true
		}
	}
	return SSHHost{}, false
}

// suggestAliases returns the SSH Aliases that contain playbookHost or are
// close to it in edit distance, nearest first then alphabetically, at most
// maxSuggestions of them.
func suggestAliases(playbookHost string, hosts []SSHHost) []string {
	type candidate struct {
		alias    string
		distance int
	}
	var candidates []candidate
	seen := map[string]bool{}
	for _, h := range hosts {
		for _, a := range h.Aliases {
			if seen[a] {
				continue
			}
			seen[a] = true
			d := editDistance(playbookHost, a)
			near := d <= maxSuggestionDistance && 2*d < len([]rune(playbookHost))
			if near || strings.Contains(a, playbookHost) {
				candidates = append(candidates, candidate{a, d})
			}
		}
	}

	slices.SortFunc(candidates, func(x, y candidate) int {
		if x.distance != y.distance {
			return x.distance - y.distance
		}
		return strings.Compare(x.alias, y.alias)
	})

	var suggestions []string
	for _, c := range candidates[:min(len(candidates), maxSuggestions)] {
		suggestions = append(suggestions, c.alias)
	}
	return suggestions
}

// editDistance is the optimal string alignment distance between a and b:
// the Levenshtein distance, with swapping two adjacent characters counted as
// a single edit.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}

func toResolved(alias string, h SSHHost, user string) ResolvedHost {
	hostname := h.HostName
	if hostname == "" {
		hostname = alias
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
