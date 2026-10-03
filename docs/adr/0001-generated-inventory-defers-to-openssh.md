# Generated Inventory defers connection settings to OpenSSH

A Generated Inventory names each entry after the SSH Alias its Playbook Host matched and records only `ansible_user`; it never writes `ansible_host`, `ansible_port` or `ansible_ssh_private_key_file`. Ansible's ssh connection plugin passes the inventory name straight to the `ssh` binary, so OpenSSH applies the operator's SSH client configuration in full (`HostName`, `Port`, `IdentityFile`, `ProxyJump`, `Include`, `Match`), whereas any `ansible_*` connection variable is passed as `-o` and overrides that configuration, so a value the tool parsed imperfectly could only make things worse. `ansible_user` is the one exception because playbooks commonly reference `{{ ansible_user }}`; its value is the user the SSH configuration effectively applies to that alias (as reported by `ssh -G`), or the Default User when the configuration names none, so it can never contradict what `ssh` would do.

## Considered Options

- Copy `User`, `Port` and `IdentityFile` from the tool's own SSH config parser into `ansible_*` variables (the original behaviour): rejected because the parser ignores wildcard, `Match` and `Include` blocks, and an `IdentityFile` passed with `-o` adds a key rather than replacing the configured ones.
- Write no `ansible_user` at all: rejected because `{{ ansible_user }}` would be undefined in playbooks that use it.
