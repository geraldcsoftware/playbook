# Playbook

A command-line tool that runs Ansible playbooks against machines the operator can already reach over SSH, without requiring an inventory file to be written for simple plays.

## Hosts

**Playbook Host**:
A value listed under a play's `hosts:` keyword, naming what that play should run against.
_Avoid_: host, host alias, target host

**SSH Host**:
One `Host` entry in the operator's SSH client configuration, together with the connection settings that apply to it.
_Avoid_: SSH config entry, server

**SSH Alias**:
One of the names on an SSH Host's `Host` line; an SSH Host may have several.
_Avoid_: alias (unqualified), hostname

**Target**:
A machine the run will act on, as determined by resolving the Playbook Hosts of every play.
_Avoid_: resolved host, node

## Inventory

**Explicit Inventory**:
An Ansible inventory the operator supplies for a run; it always takes precedence over a Generated Inventory.
_Avoid_: user inventory, custom inventory

**Generated Inventory**:
A temporary inventory the tool builds from the SSH client configuration when no Explicit Inventory is supplied.
_Avoid_: temp inventory, auto inventory
