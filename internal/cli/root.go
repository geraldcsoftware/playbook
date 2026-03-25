package cli

import (
	"fmt"
	"time"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/internal/tui"
	"github.com/geraldcsoftware/playbook/pkg/playbook"
	"github.com/geraldcsoftware/playbook/pkg/ssh"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	verbose     bool
	noPreflight bool
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playbook",
		Short: "CLI tool for running Ansible playbooks",
		Long:  "Automates Ansible playbook execution with SSH pre-flight checks, credential injection via Bitwarden AAC, and an interactive TUI.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			cfg, _ := config.Load(configFilePath())

			pb, err := playbook.Parse(args[0])
			if err != nil {
				return err
			}

			sshHosts, err := ssh.ParseConfig(sshConfigPath())
			if err != nil {
				return fmt.Errorf("parsing SSH config: %w", err)
			}

			var resolved []ssh.ResolvedHost
			var resolveErrors []string
			for _, hostAlias := range pb.Hosts {
				r, err := ssh.Resolve(hostAlias, sshHosts, cfg.DefaultUser)
				if err != nil {
					resolveErrors = append(resolveErrors, fmt.Sprintf("%s: %v", hostAlias, err))
					continue
				}
				resolved = append(resolved, r...)
			}

			action, err := tui.Run(pb, resolved, resolveErrors)
			if err != nil {
				return err
			}

			switch action {
			case tui.ActionRun:
				return runPlaybook(args[0], nil, 30*time.Second)
			case tui.ActionDoctor:
				return runDoctor()
			case tui.ActionViewHosts:
				for _, r := range resolved {
					fmt.Printf("  %s -> %s (user: %s, key: %s, port: %d)\n",
						r.Alias, r.Hostname, r.User, r.IdentityFile, r.Port)
				}
			case tui.ActionQuit:
				// nothing
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/playbook/config.yaml)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "increase output detail")
	cmd.PersistentFlags().BoolVar(&noPreflight, "no-preflight", false, "skip SSH reachability checks")

	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newHostsCmd())
	cmd.AddCommand(newRunCmd())

	return cmd
}

func Execute() error {
	return newRootCmd().Execute()
}
