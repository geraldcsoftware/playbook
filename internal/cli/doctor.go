package cli

import (
	"fmt"
	"os"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/pkg/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate toolchain and configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	fmt.Println("\033[36m◇\033[0m  \033[1m\033[97mDoctor\033[0m")

	checks := []doctor.Check{
		withHint(doctor.CheckBinary("ansible-playbook"), "https://docs.ansible.com/ansible/latest/installation_guide/"),
		withHint(doctor.CheckBinary("aac"), "https://github.com/bitwarden/agent-access"),
		withHint(doctor.CheckProcessRunning("aac listen", "aac listen"),
			"Run 'aac listen' in a separate terminal.\n"+
				"\033[2m\033[90m│\033[0m    Ensure your Bitwarden vault is unlocked first: bw unlock\n"+
				"\033[2m\033[90m│\033[0m    Then start the listener: aac listen"),
		asOptional(withHint(doctor.CheckBinary("bw"), "Required by aac — https://bitwarden.com/help/cli/")),
		doctor.CheckBinary("ssh-keygen"),
		withHint(doctor.CheckBinary("ssh-copy-id"), "brew install openssh"),
		doctor.CheckFile(sshConfigPath(), "SSH config"),
		doctor.CheckFile(configFilePath(), "playbook config"),
		withHint(doctor.CheckEnvVar("BW_EUS_ITEM_ID"), "export BW_EUS_ITEM_ID=<your-bitwarden-item-id>"),
	}

	allOK := true
	for _, c := range checks {
		printCheck(c)
		if !c.OK && !c.Optional {
			allOK = false
		}
	}

	if allOK {
		fmt.Println("\033[32m■\033[0m  \033[97mAll checks passed\033[0m")
		return nil
	}

	fmt.Println("\033[31m■\033[0m  \033[31mSome checks failed\033[0m")
	return fmt.Errorf("doctor: some checks failed")
}

func printCheck(c doctor.Check) {
	if c.OK {
		fmt.Printf("\033[2m\033[90m│\033[0m  %-28s \033[32m✓\033[0m  %s\n", c.Name, c.Detail)
	} else {
		fmt.Printf("\033[2m\033[90m│\033[0m  %-28s \033[31m✗\033[0m  not found\n", c.Name)
		if c.Hint != "" {
			fmt.Printf("\033[2m\033[90m│\033[0m    %s\n", c.Hint)
		}
	}
}

func withHint(c doctor.Check, hint string) doctor.Check {
	if c.Hint == "" {
		c.Hint = hint
	}
	return c
}

func asOptional(c doctor.Check) doctor.Check {
	c.Optional = true
	return c
}

func sshConfigPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.ssh/config"
}

func configFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultPath()
}
