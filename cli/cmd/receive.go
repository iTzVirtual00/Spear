package cmd

import (
	"spear/config"

	"github.com/spf13/cobra"
)

var receiveCmd = &cobra.Command{
	Use:   "receive",
	Short: "Accept incoming file uploads",
	Run: func(cmd *cobra.Command, args []string) {
		// receiveParams.Port, receiveParams.Single, etc.
	},
}

var receiveParams = config.ReceiveParams{}

func init() {
	receiveCmd.Flags().IntVarP(&receiveParams.Port, "port", "p", 9000, "Port to listen on for incoming files")

	receiveCmd.Flags().BoolVarP(&receiveParams.Single, "single", "s", false, "Accept a single file then exit")
	receiveCmd.Flags().BoolVarP(&receiveParams.Batch, "batch", "b", true, "Keep accepting files until stopped")

	receiveCmd.Flags().StringVarP(&receiveParams.Dest, "dest", "d", ".", "Directory to save incoming files")
	receiveCmd.Flags().StringArrayVarP(&receiveParams.AllowIDs, "allow", "a", []string{}, "List of sender IDs allowed to upload")
	receiveCmd.Flags().BoolVarP(&receiveParams.NatPunch, "nat-punch", "n", false, "Enable NAT punchthrough advertisement")

	rootCmd.AddCommand(receiveCmd)
}
