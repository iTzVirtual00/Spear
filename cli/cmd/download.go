package cmd

import (
	"spear/config"
	"spear/handlers/download"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <peer-address> <out-dir> [<file>...]",
	Short: "Download files from a sharing peer",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		downloadParams.PeerAddress = args[0]
		downloadParams.OutDir = args[1]
		downloadParams.Files = args[2:]
		download.SpearDownload(&spearConfig, &downloadParams)
	},
}

var downloadParams = config.DownloadParams{}

func init() {
	downloadCmd.Flags().StringVar(&downloadParams.WithID, "with-id", "", "Client identifier for access control")
	downloadCmd.Flags().StringVar(&downloadParams.FromID, "from-id", "", "Other peer identifier (empty=no check)")

	downloadCmd.Flags().BoolVarP(&downloadParams.NatPunch, "nat-punch", "n", false, "Use NAT punchthrough")

	rootCmd.AddCommand(downloadCmd)
}
