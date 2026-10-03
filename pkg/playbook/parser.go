package playbook

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Playbook is a playbook file as Host Resolution sees it: every play it
// declares, in file order.
type Playbook struct {
	// Name is the first play's name, used when the playbook is displayed.
	Name  string
	File  string
	Plays []Play
	// Imports lists the playbooks named by import_playbook entries, which
	// are not plays and are not read.
	Imports []string
}

// Play is one play declared in a playbook, with its Playbook Hosts in the
// order they are listed.
type Play struct {
	Name  string
	Hosts []string
}

// Hosts returns the Playbook Hosts of every play, in the order they first
// appear, each named once however many plays list it.
func (p Playbook) Hosts() []string {
	var hosts []string
	seen := map[string]bool{}
	for _, play := range p.Plays {
		for _, h := range play.Hosts {
			if !seen[h] {
				seen[h] = true
				hosts = append(hosts, h)
			}
		}
	}
	return hosts
}

// importKeys are the keys that mark a playbook entry as an import_playbook
// rather than a play.
var importKeys = []string{"import_playbook", "ansible.builtin.import_playbook"}

// Parse reads a playbook, returning its Playbook Hosts as written. It does
// not judge whether Host Resolution can match them: patterns, templates and
// imports are accepted, since an Explicit Inventory may supply the hosts,
// and Host Resolution reports them otherwise.
func Parse(path string) (Playbook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Playbook{}, fmt.Errorf("reading playbook: %w", err)
	}

	var raw []map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Playbook{}, fmt.Errorf("parsing playbook YAML: %w", err)
	}

	pb := Playbook{File: path}
	for _, entry := range raw {
		if target, ok := importTarget(entry); ok {
			pb.Imports = append(pb.Imports, target)
			continue
		}
		name, _ := entry["name"].(string)
		hosts, err := extractHosts(entry["hosts"])
		if err != nil {
			return Playbook{}, err
		}
		pb.Plays = append(pb.Plays, Play{Name: name, Hosts: hosts})
	}

	if len(pb.Plays) == 0 && len(pb.Imports) == 0 {
		return Playbook{}, fmt.Errorf("playbook contains no plays")
	}
	if len(pb.Plays) > 0 {
		pb.Name = pb.Plays[0].Name
	}
	return pb, nil
}

// importTarget reports whether entry is an import_playbook rather than a
// play, and if so the playbook it names.
func importTarget(entry map[string]interface{}) (string, bool) {
	if _, ok := entry["hosts"]; ok {
		return "", false
	}
	for _, k := range importKeys {
		if v, ok := entry[k]; ok {
			target, _ := v.(string)
			return target, true
		}
	}
	return "", false
}

func extractHosts(v interface{}) ([]string, error) {
	if v == nil {
		return nil, fmt.Errorf("playbook has no 'hosts' field")
	}

	var values []string
	switch val := v.(type) {
	case string:
		values = []string{val}
	case []interface{}:
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("hosts list contains non-string value: %v", item)
			}
			values = append(values, s)
		}
	default:
		return nil, fmt.Errorf("unsupported hosts type: %T", v)
	}

	var hosts []string
	for _, value := range values {
		hosts = append(hosts, splitHosts(value)...)
	}
	return hosts, nil
}

// splitHosts splits a hosts value such as "web, db" on commas, as Ansible
// does, dropping surrounding spaces and empty parts. A value that is
// templated, or that holds an Ansible host pattern, is kept whole, so that
// it is reported as the operator wrote it rather than in fragments.
func splitHosts(value string) []string {
	if IsTemplate(value) || IsPattern(value) {
		return []string{strings.TrimSpace(value)}
	}
	return hostParts(value)
}

// hostParts splits value on commas, dropping surrounding spaces and empty
// parts.
func hostParts(value string) []string {
	var parts []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// patternChars are the characters that make a Playbook Host an Ansible host
// pattern (a wildcard, a group operator, a regular expression or a range)
// rather than a name an SSH Alias could equal.
const patternChars = "*?:&!~[]"

// IsTemplate reports whether a Playbook Host is a Jinja template, such as
// "{{ target }}", whose value is known only when the play runs.
func IsTemplate(host string) bool {
	return strings.Contains(host, "{{")
}

// IsPattern reports whether a Playbook Host is an Ansible host pattern
// rather than a name: any of its comma-separated parts is "all" or contains
// a wildcard, a group operator, a regular expression marker or a range.
func IsPattern(host string) bool {
	for _, part := range hostParts(host) {
		if part == "all" || strings.ContainsAny(part, patternChars) {
			return true
		}
	}
	return false
}
