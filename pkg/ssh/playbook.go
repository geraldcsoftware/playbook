package ssh

import (
	"slices"

	"github.com/geraldcsoftware/playbook/pkg/playbook"
)

// ResolvePlaybook performs Host Resolution for every play of pb, as Resolve
// does for its Playbook Hosts, and also reports what Host Resolution cannot
// handle without an Explicit Inventory: each import_playbook entry, each
// Playbook Host that is an Ansible host pattern or a template, and each play
// that names one SSH Host by two of its SSH Aliases. Patterns and templates
// are not looked up. Every failure is returned together, imports first,
// then those of each Playbook Host, then repeated SSH Hosts; a run must not
// proceed while any exists.
func ResolvePlaybook(pb playbook.Playbook, config Config, lookup EffectiveSettingsLookup, defaultUser string) ([]ResolvedHost, []HostResolutionFailure) {
	var failures []HostResolutionFailure
	for _, imp := range pb.Imports {
		failures = append(failures, HostResolutionFailure{Kind: ImportedPlaybook, Import: imp})
	}

	var names []string
	for _, ph := range pb.Hosts() {
		switch {
		case playbook.IsTemplate(ph):
			failures = append(failures, HostResolutionFailure{Kind: TemplatedHost, PlaybookHost: ph})
		case playbook.IsPattern(ph):
			failures = append(failures, HostResolutionFailure{Kind: HostPattern, PlaybookHost: ph})
		default:
			names = append(names, ph)
		}
	}

	targets, unresolved := Resolve(names, config, lookup, defaultUser)
	failures = append(failures, unresolved...)
	failures = append(failures, repeatedSSHHosts(pb.Plays, config.Hosts)...)
	return targets, failures
}

// repeatedSSHHosts reports each Playbook Host of a play that is an SSH Alias
// of the same SSH Host as one listed earlier in that play. The same SSH Host
// named in different plays is allowed.
func repeatedSSHHosts(plays []playbook.Play, hosts []SSHHost) []HostResolutionFailure {
	var failures []HostResolutionFailure
	for i, play := range plays {
		first := map[int]string{}
		for _, ph := range play.Hosts {
			h := slices.IndexFunc(hosts, func(h SSHHost) bool { return slices.Contains(h.Aliases, ph) })
			if h < 0 {
				continue
			}
			earlier, seen := first[h]
			switch {
			case !seen:
				first[h] = ph
			case earlier != ph:
				failures = append(failures, HostResolutionFailure{
					Kind:          SSHHostRepeatedInPlay,
					PlaybookHost:  ph,
					SameSSHHostAs: earlier,
					Play:          play.Name,
					PlayNumber:    i + 1,
				})
			}
		}
	}
	return failures
}
