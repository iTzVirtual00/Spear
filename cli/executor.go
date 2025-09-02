package cli

import (
	"spear/cli/cmd"
	"spear/config"
)

func RunCLI(config config.SpearConfig) {
	errCmd := cmd.Execute(config)
	if errCmd != nil {
		return
	}
}
