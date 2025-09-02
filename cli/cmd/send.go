package cmd

import (
	"spear/config"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send <peer-address> <file> [<file>...]",
	Short: "Send files to a listening peer",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sendParams.PeerAddress = args[0]
		// files := args[1:]
		// sendParams.Port, sendParams.FromID, sendParams.NatPunch
		// files contains the list of files to send
	},
}

var sendParams = config.SendParams{}

func init() {
	sendCmd.Flags().IntVarP(&sendParams.Port, "port", "p", 9000, "Port to connect to on the peer")
	sendCmd.Flags().StringVar(&sendParams.FromID, "from-id", "", "Sender identifier for access control")
	sendCmd.Flags().BoolVarP(&sendParams.NatPunch, "nat-punch", "n", false, "Use NAT punchthrough")

	rootCmd.AddCommand(sendCmd)
}
