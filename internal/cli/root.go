package cli

import (
	"fmt"

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
			// args[0] is a playbook file — launch TUI (implemented in a later task)
			fmt.Printf("TUI not yet implemented. Use 'playbook run %s' instead.\n", args[0])
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/playbook/config.yaml)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "increase output detail")
	cmd.PersistentFlags().BoolVar(&noPreflight, "no-preflight", false, "skip SSH reachability checks")

	cmd.AddCommand(newDoctorCmd())

	return cmd
}

func Execute() error {
	return newRootCmd().Execute()
}
