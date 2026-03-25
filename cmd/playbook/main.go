package main

import (
	"os"

	"github.com/geraldcsoftware/playbook/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
