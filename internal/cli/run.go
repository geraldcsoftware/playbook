package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/pkg/ansible"
	"github.com/geraldcsoftware/playbook/pkg/credentials"
	"github.com/geraldcsoftware/playbook/pkg/inventory"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
	"github.com/spf13/cobra"
)

var (
	credentialProvider string
	secretName         string
	accessToken        string
)

func newRunCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "run <playbook.yml> [-- extra-ansible-args...]",
		Short: "Run an Ansible playbook with pre-flight checks and credential injection",
		Long: `Run an Ansible playbook with pre-flight checks and credential injection.

Without --inventory, each Playbook Host is matched to an SSH Alias in ~/.ssh/config
and a Generated Inventory is passed to ansible-playbook. With --inventory (-i), that
Explicit Inventory is passed instead, and Host Resolution and the SSH pre-flight
check are skipped. Either way the 'inventory' setting in ansible.cfg is ignored.

Supply the inventory only through --inventory: -i or --inventory after '--', or in
ansible.default_args in the config, stops the run.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			playbookFile := args[0]
			var extraArgs []string
			if cmd.ArgsLenAtDash() > 0 {
				extraArgs = args[cmd.ArgsLenAtDash():]
			}

			return runPlaybook(playbookFile, extraArgs, time.Duration(timeout)*time.Second)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 30, "SSH pre-flight timeout in seconds")
	cmd.Flags().StringVarP(&credentialProvider, "credential-provider", "p", "", "credential provider: aac or bws (overrides config)")
	cmd.Flags().StringVarP(&secretName, "secret-name", "s", "", "BWS secret name (overrides config)")
	cmd.Flags().StringVarP(&accessToken, "access-token", "t", "", "BWS access token (overrides config env var)")

	return cmd
}

func resolveProvider(cfg config.Config) (credentials.Provider, error) {
	providerName := cfg.CredentialProvider
	if credentialProvider != "" {
		providerName = credentialProvider
	}

	switch providerName {
	case "aac":
		itemID := os.Getenv(cfg.AAC.ItemIDEnv)
		if itemID == "" {
			return nil, fmt.Errorf("$%s not set — run 'playbook doctor' to diagnose", cfg.AAC.ItemIDEnv)
		}
		return credentials.NewAACProvider(itemID), nil

	case "bws":
		token := accessToken
		if token == "" {
			token = os.Getenv(cfg.BWS.AccessTokenEnv)
		}
		if token == "" {
			return nil, fmt.Errorf("BWS access token not set — pass --access-token or set $%s", cfg.BWS.AccessTokenEnv)
		}

		name := secretName
		if name == "" {
			name = cfg.BWS.SecretName
		}
		if name == "" {
			return nil, fmt.Errorf("BWS secret name not set — pass --secret-name or set bws.secret_name in config")
		}

		return credentials.NewBWSProvider(token, name), nil

	default:
		return nil, fmt.Errorf("unknown credential provider '%s' — use 'aac' or 'bws'", providerName)
	}
}

func runPlaybook(playbookFile string, extraArgs []string, timeout time.Duration) error {
	cfg, _ := config.Load(configFilePath())

	if err := checkNoInventoryArgs(cfg.Ansible.DefaultArgs, extraArgs); err != nil {
		return err
	}

	// Phase 1: Playbook Discovery
	fmt.Println("\033[36m◇\033[0m  \033[1m\033[97mPlaybook Discovery\033[0m")

	pb, err := playbook.Parse(playbookFile)
	if err != nil {
		return err
	}
	fmt.Printf("\033[32m■\033[0m  Found: %s\n", pb.Name)

	// Phase 2: Host Resolution
	fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mHost Resolution\033[0m")

	var allResolved []ssh.ResolvedHost
	if explicitInventory != "" {
		fmt.Printf("\033[2m\033[90m│\033[0m  Skipped: Explicit Inventory %s supplies the hosts\n", explicitInventory)
	} else {
		targets, failures, err := resolveHosts(pb, cfg)
		if err != nil {
			return err
		}
		allResolved = targets

		for _, r := range targets {
			fmt.Printf("\033[2m\033[90m│\033[0m  %s → %s\n", r.Alias, r.Hostname)
		}
		for _, f := range failures {
			fmt.Printf("\033[2m\033[90m│\033[0m  %s — \033[31m✗\033[0m %s\n", f.Subject(), f.Error())
		}
		if len(failures) > 0 {
			fmt.Println("\033[31m■\033[0m  \033[31mHost Resolution failed\033[0m")
			return fmt.Errorf("Host Resolution failed with %d problem(s)", len(failures))
		}
		fmt.Printf("\033[2m\033[90m│\033[0m  \033[97m%d host(s) resolved\033[0m \033[32m✓\033[0m\n", len(allResolved))
	}

	// Phase 3: SSH Pre-flight
	if explicitInventory != "" {
		fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mSSH Pre-flight\033[0m")
		fmt.Println("\033[2m\033[90m│\033[0m  Skipped: hosts come from the Explicit Inventory, not the SSH config")
	} else if !noPreflight {
		fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mSSH Pre-flight\033[0m")

		results := ssh.RunPreflight(allResolved, timeout)
		for _, r := range results {
			if r.Reachable {
				fmt.Printf("\033[2m\033[90m│\033[0m  %s (%s) — reachable \033[32m✓\033[0m\n", r.Alias, r.Address())
			} else {
				fmt.Printf("\033[2m\033[90m│\033[0m  %s (%s) — \033[31m✗\033[0m %s\n", r.Alias, r.Address(), r.Error)
			}
		}

		if !ssh.AllPassed(results) {
			fmt.Println("\033[31m■\033[0m  \033[31mPre-flight failed\033[0m")
			return fmt.Errorf("SSH pre-flight failed")
		}
		fmt.Printf("\033[32m■\033[0m  All hosts passed\n")
	}

	// Phase 4: Resolve credential provider
	fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mCredential Injection\033[0m")

	provider, err := resolveProvider(cfg)
	if err != nil {
		return err
	}

	providerName := cfg.CredentialProvider
	if credentialProvider != "" {
		providerName = credentialProvider
	}
	fmt.Printf("\033[2m\033[90m│\033[0m  Provider: %s\n", providerName)

	// Phase 5: Generate inventory, unless an Explicit Inventory is in use
	invPath := explicitInventory
	if invPath == "" {
		generated, cleanup, err := inventory.Generate(allResolved)
		if err != nil {
			return fmt.Errorf("generating inventory: %w", err)
		}
		defer cleanup()
		invPath = generated
	}

	// Phase 6: Run ansible-playbook
	fmt.Println("\n\033[32m■\033[0m  \033[32mHanding off to ansible-playbook...\033[0m")
	fmt.Println("\033[2m\033[90m" + "──────────────────────────────────────────────────" + "\033[0m")
	fmt.Println()

	allExtraArgs := append(cfg.Ansible.DefaultArgs, extraArgs...)

	runner := ansible.NewRunner(provider)
	exitCode, err := runner.Run(playbookFile, invPath, allExtraArgs)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("ansible-playbook exited with code %d", exitCode)
	}

	return nil
}
