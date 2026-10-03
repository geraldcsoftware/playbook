# SSH Hosts are managed outside playbook

playbook only reads the operator's SSH client configuration; it does not create SSH Hosts, generate or install keys, or edit `~/.ssh/config`. Provisioning SSH access is owned by whatever process already governs it for those machines, and a tool that silently appends `Host` blocks and passphrase-less keys competes with that process. `playbook hosts add` is therefore deprecated, with a warning on every use, and will be removed in the following release.

## Considered Options

- Extend `hosts add` with repeatable `--alias` flags and duplicate detection, so short Playbook Hosts work under exact matching: rejected as out of scope for a playbook runner.
