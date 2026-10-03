package ssh

import (
	"fmt"
	"slices"
	"strings"
)

// ResolvedHost is a Target. Hostname and Port are the address and port
// OpenSSH effectively connects to for Alias, as `ssh -G` reports them.
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

// FailureKind says why Host Resolution could not produce a run's Targets.
type FailureKind int

const (
	// NoMatchingAlias: a Playbook Host equals no SSH Alias; Suggestions
	// lists the nearest SSH Aliases, nearest first.
	NoMatchingAlias FailureKind = iota
	// SettingsUnreadable: a Playbook Host matched an SSH Alias whose
	// effective settings could not be read; Err says why.
	SettingsUnreadable
	// HostPattern: a Playbook Host is an Ansible host pattern, such as
	// "all" or "web:&prod", rather than a name.
	HostPattern
	// TemplatedHost: a Playbook Host is a template, such as "{{ target }}",
	// whose value is known only when the play runs.
	TemplatedHost
	// ImportedPlaybook: the playbook imports Import with import_playbook,
	// whose plays Host Resolution does not read.
	ImportedPlaybook
	// SSHHostRepeatedInPlay: PlaybookHost and SameSSHHostAs, both listed in
	// Play, are SSH Aliases of one SSH Host.
	SSHHostRepeatedInPlay
)

// HostResolutionFailure is one reason Host Resolution could not turn a
// playbook into Targets; Kind says which, and so which other fields are set.
// A run must not proceed while any failure exists.
type HostResolutionFailure struct {
	Kind         FailureKind
	PlaybookHost string
	Suggestions  []string
	Err          error
	// Import is the imported playbook, for ImportedPlaybook.
	Import string
	// Play is the play's name, if it has one, and PlayNumber its position
	// in the playbook, counting from one; SameSSHHostAs is the Playbook
	// Host listed before PlaybookHost. All are for SSHHostRepeatedInPlay.
	Play          string
	PlayNumber    int
	SameSSHHostAs string
}

// Subject names what failed, for display ahead of the Error message: the
// Playbook Host or Hosts concerned, or the import_playbook entry.
func (f HostResolutionFailure) Subject() string {
	switch f.Kind {
	case ImportedPlaybook:
		return "import_playbook " + f.Import
	case SSHHostRepeatedInPlay:
		return f.SameSSHHostAs + ", " + f.PlaybookHost
	default:
		return f.PlaybookHost
	}
}

func (f HostResolutionFailure) Error() string {
	switch f.Kind {
	case SettingsUnreadable:
		return fmt.Sprintf("could not read the SSH settings for SSH Alias '%s': %v — fix its SSH configuration or pass --inventory", f.PlaybookHost, f.Err)
	case HostPattern:
		return fmt.Sprintf("'%s' is an Ansible host pattern, which Host Resolution cannot match to an SSH Alias — name SSH Aliases, or pass --inventory to run against an Explicit Inventory", f.PlaybookHost)
	case TemplatedHost:
		return fmt.Sprintf("'%s' is templated, so its value is not known until the play runs — name SSH Aliases, or pass --inventory to run against an Explicit Inventory", f.PlaybookHost)
	case ImportedPlaybook:
		return fmt.Sprintf("the playbook imports '%s', and imported playbooks need --inventory for now", f.Import)
	case SSHHostRepeatedInPlay:
		play := fmt.Sprintf("%d", f.PlayNumber)
		if f.Play != "" {
			play = fmt.Sprintf("'%s'", f.Play)
		}
		return fmt.Sprintf("'%s' and '%s' in play %s are SSH Aliases of the same SSH Host — list it once in the play, or pass --inventory", f.SameSSHHostAs, f.PlaybookHost, play)
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
			failures = append(failures, HostResolutionFailure{Kind: SettingsUnreadable, PlaybookHost: ph, Err: err})
			continue
		}
		user := defaultUser
		if config.SetsUser(ph) && settings.User() != "" {
			user = settings.User()
		}
		targets = append(targets, toResolved(ph, h, settings, user))
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

// toResolved builds the Target for alias. Its address and port are the ones
// OpenSSH effectively applies, so a Port set only in a wildcard block is
// honoured; where the settings lack them it falls back to OpenSSH's own
// defaults, the alias itself and port 22.
func toResolved(alias string, h SSHHost, settings EffectiveSettings, user string) ResolvedHost {
	hostname := settings.HostName()
	if hostname == "" {
		hostname = alias
	}
	port := settings.Port()
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
