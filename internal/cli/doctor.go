package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/geraldcsoftware/playbook/internal/config"
	"github.com/geraldcsoftware/playbook/pkg/credentials"
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
	cfg, cfgErr := config.Load(configFilePath())

	providerName := cfg.CredentialProvider
	if credentialProvider != "" {
		providerName = credentialProvider
	}

	fmt.Println("\033[36m◇\033[0m  \033[1m\033[97mDoctor\033[0m")

	// Common checks
	checks := []doctor.Check{
		withHint(doctor.CheckBinary("ansible-playbook"), "https://docs.ansible.com/ansible/latest/installation_guide/"),
		doctor.CheckFile(sshConfigPath(), "SSH config"),
		checkConfigFile(configFilePath(), cfgErr),
	}

	// Provider-specific checks
	switch providerName {
	case "aac":
		checks = append(checks,
			withHint(doctor.CheckBinary("aac"), "https://github.com/bitwarden/agent-access"),
			withHint(doctor.CheckProcessRunning("aac listen", "aac listen"),
				"Run 'aac listen' in a separate terminal.\n"+
					"\033[2m\033[90m│\033[0m    Ensure your Bitwarden vault is unlocked first: bw unlock\n"+
					"\033[2m\033[90m│\033[0m    Then start the listener: aac listen"),
			asOptional(withHint(doctor.CheckBinary("bw"), "Required by aac — https://bitwarden.com/help/cli/")),
			withHint(doctor.CheckEnvVar(cfg.AAC.ItemIDEnv), fmt.Sprintf("export %s=<your-bitwarden-item-id>", cfg.AAC.ItemIDEnv)),
		)
	case "bws":
		token := os.Getenv(cfg.BWS.AccessTokenEnv)
		checks = append(checks,
			withHint(doctor.CheckBinary("bws"), "https://bitwarden.com/help/secrets-manager-cli/"),
			withHint(doctor.CheckEnvVar(cfg.BWS.AccessTokenEnv), fmt.Sprintf("export %s=<your-bws-access-token>", cfg.BWS.AccessTokenEnv)),
		)
		if token != "" {
			checks = append(checks,
				withHint(doctor.CheckCommand("bws", "bws auth", credentials.BWSArgs(token, "secret", "list")...),
					"Check that your BWS access token is valid and has the right permissions"),
			)
			if cfg.BWS.SecretName != "" {
				checks = append(checks, doctor.Check{
					Name:   "bws secret '" + cfg.BWS.SecretName + "'",
					OK:     true,
					Detail: "configured",
				})
			}
		}
	}

	allOK := true
	for _, c := range checks {
		printCheck(c)
		if !c.OK && !c.Optional && !c.Warning {
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

// checkConfigFile reports on the playbook config file from the error
// config.Load returned for it. A missing or unusable file only warns: every
// command still runs on the built-in defaults.
func checkConfigFile(path string, loadErr error) doctor.Check {
	switch {
	case loadErr == nil:
		return doctor.Check{Name: path, OK: true, Detail: "playbook config"}
	case errors.Is(loadErr, config.ErrNotFound):
		return doctor.Check{
			Name:    path,
			Warning: true,
			Detail:  "no playbook config file; using built-in defaults",
			Hint: "Create it to set default_user, the account a Target is connected as\n" +
				"\033[2m\033[90m│\033[0m    when its SSH Host names none.",
		}
	default:
		return doctor.Check{
			Name:    path,
			Warning: true,
			Detail:  strings.Join(strings.Fields(loadErr.Error()), " ") + "; using built-in defaults",
			Hint:    "Fix or remove the file so that your settings take effect.",
		}
	}
}

func printCheck(c doctor.Check) {
	if c.Warning {
		fmt.Printf("\033[2m\033[90m│\033[0m  %-28s \033[33m◇\033[0m  \033[33mWARNING:\033[0m %s\n", c.Name, c.Detail)
		if c.Hint != "" {
			fmt.Printf("\033[2m\033[90m│\033[0m    %s\n", c.Hint)
		}
	} else if c.OK {
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
