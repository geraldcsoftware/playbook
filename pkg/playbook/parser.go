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

// Parse reads a playbook for Host Resolution, rejecting Playbook Hosts that
// are Ansible patterns rather than names an SSH Alias could match.
func Parse(path string) (Playbook, error) {
	return parse(path, validateHostPattern)
}

// ParseAnyPattern reads a playbook whose hosts come from an Explicit
// Inventory, so any Ansible host pattern is accepted.
func ParseAnyPattern(path string) (Playbook, error) {
	return parse(path, func(string) error { return nil })
}

func parse(path string, validate func(string) error) (Playbook, error) {
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
		hosts, err := extractHosts(entry["hosts"], validate)
		if err != nil {
			return Playbook{}, err
		}
		pb.Plays = append(pb.Plays, Play{Name: name, Hosts: hosts})
	}

	if len(pb.Plays) == 0 {
		return Playbook{}, fmt.Errorf("playbook contains no plays")
	}
	pb.Name = pb.Plays[0].Name
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

func extractHosts(v interface{}, validate func(string) error) ([]string, error) {
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
		for _, h := range splitHosts(value) {
			if err := validate(h); err != nil {
				return nil, err
			}
			hosts = append(hosts, h)
		}
	}
	return hosts, nil
}

// splitHosts splits a hosts value such as "web, db" on commas, as Ansible
// does, dropping surrounding spaces and empty parts.
func splitHosts(value string) []string {
	var hosts []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			hosts = append(hosts, part)
		}
	}
	return hosts
}

func validateHostPattern(host string) error {
	if host == "all" {
		return fmt.Errorf("host pattern 'all' is not supported — this tool resolves individual hosts from ~/.ssh/config")
	}
	for _, ch := range []string{":", "&", "!", "*"} {
		if strings.Contains(host, ch) {
			return fmt.Errorf("host pattern '%s' contains '%s' — Ansible patterns are not supported, use explicit hostnames", host, ch)
		}
	}
	return nil
}
