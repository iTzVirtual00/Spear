package cmd

import (
	"github.com/spf13/cobra"
)

var utilsCmd = &cobra.Command{
	Use:   "utils",
	Short: "Utility commands",
}

func init() {
	rootCmd.AddCommand(utilsCmd)
	utilsCmd.AddCommand(fingerprintCmd)
}
