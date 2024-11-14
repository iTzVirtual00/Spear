package cmd

import (
	"github.com/spf13/cobra"
	"spear/entrypoints"
)

var (
	sendCommand = entrypoints.SendCommand{
		Listen: DefaultListenValue, // default
	}
	sendCmd = &cobra.Command{
		Use: "send",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("address").Changed {
				// spear send -a Address[:port] -f <outfile>
				entrypoints.SendClient(&spearConfig, &sendCommand)
			} else {
				// spear send -l Port -f <outfile>
				entrypoints.SendAsServer(&spearConfig, &sendCommand)
			}
		},
	}
)

func init() {
	sendCmd.Flags().VarP(&sendCommand.Address, "address", "a", "asd")
	sendCmd.Flags().VarP(&sendCommand.Listen, "listen", "l", "asd")
	sendCmd.MarkFlagsMutuallyExclusive("listen", "address")

	sendCmd.Flags().StringArrayVarP(&sendCommand.Files, "file", "f", []string{}, "asd")
	sendCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(sendCmd)
}
