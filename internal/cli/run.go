package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/pkg/ansible"
	"github.com/geraldcsoftware/playbook/pkg/credentials"
	"github.com/geraldcsoftware/playbook/pkg/inventory"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "run <playbook.yml> [-- extra-ansible-args...]",
		Short: "Run an Ansible playbook with pre-flight checks and credential injection",
		Args:  cobra.MinimumNArgs(1),
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

	return cmd
}

func runPlaybook(playbookFile string, extraArgs []string, timeout time.Duration) error {
	cfg, _ := config.Load(configFilePath())

	// Phase 1: Playbook Discovery
	fmt.Println("\033[36m◇\033[0m  \033[1m\033[97mPlaybook Discovery\033[0m")

	pb, err := playbook.Parse(playbookFile)
	if err != nil {
		return err
	}
	fmt.Printf("\033[32m■\033[0m  Found: %s\n", pb.Name)

	// Phase 2: Host Resolution
	fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mHost Resolution\033[0m")

	sshHosts, err := ssh.ParseConfig(sshConfigPath())
	if err != nil {
		return fmt.Errorf("parsing SSH config: %w", err)
	}

	var allResolved []ssh.ResolvedHost
	for _, hostAlias := range pb.Hosts {
		resolved, err := ssh.Resolve(hostAlias, sshHosts, cfg.DefaultUser)
		if err != nil {
			return fmt.Errorf("resolving host '%s': %w", hostAlias, err)
		}
		allResolved = append(allResolved, resolved...)
		for _, r := range resolved {
			fmt.Printf("\033[2m\033[90m│\033[0m  %s → %s\n", hostAlias, r.Hostname)
		}
	}
	fmt.Printf("\033[2m\033[90m│\033[0m  \033[97m%d host(s) resolved\033[0m \033[32m✓\033[0m\n", len(allResolved))

	// Phase 3: SSH Pre-flight
	if !noPreflight {
		fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mSSH Pre-flight\033[0m")

		results := ssh.RunPreflight(allResolved, timeout)
		for _, r := range results {
			if r.Reachable {
				fmt.Printf("\033[2m\033[90m│\033[0m  %s — port %d reachable \033[32m✓\033[0m\n", r.Host, r.Port)
			} else {
				fmt.Printf("\033[2m\033[90m│\033[0m  %s — \033[31m✗\033[0m %s\n", r.Host, r.Error)
			}
		}

		if !ssh.AllPassed(results) {
			fmt.Println("\033[31m■\033[0m  \033[31mPre-flight failed\033[0m")
			return fmt.Errorf("SSH pre-flight failed")
		}
		fmt.Printf("\033[32m■\033[0m  All hosts passed\n")
	}

	// Phase 4: Credential check
	fmt.Println("\n\033[36m◇\033[0m  \033[1m\033[97mCredential Injection\033[0m")

	itemID := os.Getenv("BW_EUS_ITEM_ID")
	if itemID == "" {
		return fmt.Errorf("$BW_EUS_ITEM_ID not set — run 'playbook doctor' to diagnose")
	}
	maskedID := itemID
	if len(itemID) > 8 {
		maskedID = itemID[:4] + "..." + itemID[len(itemID)-4:]
	}
	fmt.Printf("\033[2m\033[90m│\033[0m  Using Bitwarden item: %s\n", maskedID)

	// Phase 5: Generate inventory
	groupName := strings.Join(pb.Hosts, "_")
	invPath, cleanup, err := inventory.Generate(groupName, allResolved)
	if err != nil {
		return fmt.Errorf("generating inventory: %w", err)
	}
	defer cleanup()

	// Phase 6: Run ansible-playbook via aac
	fmt.Println("\n\033[32m■\033[0m  \033[32mHanding off to ansible-playbook...\033[0m")
	fmt.Println("\033[2m\033[90m" + "──────────────────────────────────────────────────" + "\033[0m")
	fmt.Println()

	allExtraArgs := append(cfg.Ansible.DefaultArgs, extraArgs...)

	provider := credentials.NewAACProvider(itemID)
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
