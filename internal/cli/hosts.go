package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
	"github.com/spf13/cobra"
)

func newHostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "Manage and inspect hosts",
	}

	cmd.AddCommand(newHostsListCmd())
	cmd.AddCommand(newHostsAddCmd())
	cmd.AddCommand(newHostsResolveCmd())

	return cmd
}

func newHostsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all hosts from ~/.ssh/config",
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, err := ssh.ParseConfig(sshConfigPath())
			if err != nil {
				return fmt.Errorf("parsing SSH config: %w", err)
			}

			if len(hosts) == 0 {
				fmt.Println("No hosts found in ~/.ssh/config")
				return nil
			}

			for _, h := range hosts {
				hostname := h.HostName
				if hostname == "" {
					hostname = h.Aliases[0]
				}
				fmt.Printf("  %s → %s (user: %s, port: %d)\n", strings.Join(h.Aliases, ", "), hostname, h.User, h.Port)
			}
			fmt.Printf("\n%d hosts found\n", len(hosts))
			return nil
		},
	}
}

func newHostsAddCmd() *cobra.Command {
	var host, user, keyType string
	var port int

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new host with SSH key setup (deprecated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			printHostsAddDeprecation(cmd.ErrOrStderr())

			cfg, _ := config.Load(configFilePath())

			if host == "" {
				fmt.Print("  Hostname: ")
				fmt.Scanln(&host)
			}
			if user == "" {
				user = cfg.EffectiveDefaultUser()
				fmt.Printf("  SSH User (%s): ", user)
				var input string
				fmt.Scanln(&input)
				if input != "" {
					user = input
				}
			}
			if port == 0 {
				port = 22
			}
			if keyType == "" {
				keyType = "ed25519"
			}

			host = strings.TrimSpace(host)
			if host == "" {
				return fmt.Errorf("hostname is required")
			}

			keyPath := ssh.KeyPath(keyType, host)
			fmt.Printf("\033[36m◇\033[0m  Generating SSH key pair...\n")
			if err := ssh.GenerateKey(keyType, keyPath); err != nil {
				return fmt.Errorf("generating key: %w", err)
			}
			fmt.Printf("\033[2m\033[90m│\033[0m  \033[32m✓\033[0m Key generated: %s\n", keyPath)

			fmt.Printf("\033[36m◇\033[0m  Copying public key to host...\n")
			if err := ssh.CopyKey(keyPath, user, host, port); err != nil {
				return fmt.Errorf("copying key: %w", err)
			}
			fmt.Printf("\033[2m\033[90m│\033[0m  \033[32m✓\033[0m Public key installed\n")

			fmt.Printf("\033[36m◇\033[0m  Updating ~/.ssh/config...\n")
			tildeKeyPath := "~/.ssh/" + keyPath[strings.LastIndex(keyPath, "/")+1:]
			entry := ssh.BuildSSHConfigEntry(host, user, tildeKeyPath, port)
			if err := ssh.AppendSSHConfigEntry(sshConfigPath(), entry); err != nil {
				return fmt.Errorf("updating ssh config: %w", err)
			}
			fmt.Printf("\033[2m\033[90m│\033[0m  \033[32m✓\033[0m Entry added\n")

			fmt.Printf("\033[36m◇\033[0m  Verifying connection...\n")
			if err := ssh.VerifyConnection(host, port, user); err != nil {
				fmt.Printf("\033[33m◇\033[0m  \033[33mWARNING:\033[0m Verification failed: %v\n", err)
				fmt.Println("  Key was installed but connection test failed. Check firewall/DNS.")
			} else {
				fmt.Printf("\033[2m\033[90m│\033[0m  \033[32m✓\033[0m SSH connection successful\n")
			}

			fmt.Printf("\033[32m■\033[0m  Host %s ready\n", host)
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "hostname (FQDN)")
	cmd.Flags().StringVar(&user, "user", "", "SSH user")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	cmd.Flags().StringVar(&keyType, "key-type", "ed25519", "SSH key type (ed25519 or rsa)")

	return cmd
}

// printHostsAddDeprecation warns that SSH hosts are to be managed outside
// playbook; the command still runs until it is removed in the next release.
func printHostsAddDeprecation(w io.Writer) {
	fmt.Fprintf(w, "\033[33m◇\033[0m  \033[33mWARNING:\033[0m 'playbook hosts add' is deprecated and will be removed in the next release.\n")
	fmt.Fprintf(w, "\033[33m│\033[0m  Manage SSH hosts directly in ~/.ssh/config, outside playbook.\n")
}

func newHostsResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <playbook.yml>",
		Short: "Show resolved hosts from a playbook file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(configFilePath())

			pb, err := playbook.Parse(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Playbook: %s\n", pb.Name)
			fmt.Printf("File:     %s\n\n", pb.File)

			if explicitInventory != "" {
				fmt.Printf("  Host Resolution skipped: Explicit Inventory %s supplies the hosts\n", explicitInventory)
				return nil
			}

			targets, failures, err := resolveHosts(pb, cfg)
			if err != nil {
				return err
			}

			for _, r := range targets {
				fmt.Printf("  ✓ %s → %s (user: %s, key: %s, port: %d)\n",
					r.Alias, r.Hostname, r.User, r.IdentityFile, r.Port)
			}
			for _, f := range failures {
				fmt.Printf("  ✗ %s — %v\n", f.Subject(), f)
			}
			if len(failures) > 0 {
				return fmt.Errorf("Host Resolution failed with %d problem(s)", len(failures))
			}
			return nil
		},
	}
}
