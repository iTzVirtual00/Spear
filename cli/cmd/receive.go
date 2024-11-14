package cmd

import (
	"github.com/spf13/cobra"
	"spear/entrypoints"
)

var (
	receiveCommand = entrypoints.ReceiveCommand{
		Listen: DefaultListenValue, // default
	}
	receiveCmd = &cobra.Command{
		Use: "receive",
		//Short: "",
		//Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("address").Changed {
				// spear receive -a Address[:port] -f <outfile>
				entrypoints.ReceiveClient(&spearConfig, &receiveCommand)
			} else {
				// spear receive -l Port -f <outfile>
				entrypoints.ReceiveServer(&spearConfig, &receiveCommand)
			}
		},
	}
)

func init() {
	receiveCmd.Flags().VarP(&receiveCommand.Address, "address", "a", "")
	receiveCmd.Flags().VarP(&receiveCommand.Listen, "listen", "l", "asd")
	receiveCmd.MarkFlagsMutuallyExclusive("listen", "address")

	receiveCmd.Flags().StringArrayVarP(&receiveCommand.Files, "file", "f", []string{}, "asd")
	receiveCmd.MarkFlagRequired("file")

	receiveCmd.Flags().StringArrayVarP(&receiveCommand.AllowedContacts, "contact", "c", []string{}, "asd")

	rootCmd.AddCommand(receiveCmd)
}
