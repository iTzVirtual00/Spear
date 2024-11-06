package cli

import (
	"spear/cli/cmd"
	"spear/config"
)

var spearConfig config.SpearConfig

func RunCLI(config config.SpearConfig) {
	spearConfig = config
	errCmd := cmd.Execute()
	if errCmd != nil {
		return
	}
}
