package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"spear/utils"

	"github.com/spf13/cobra"
)

type FingerprintParams struct {
	CertFile string // path to certificate file (PEM or DER)
}

var fingerprintCmd = &cobra.Command{
	Use:   "calculate-fingerprint <cert-file>",
	Short: "Calculate SHA-256 fingerprint of a certificate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}

		sum := utils.GetCertFingerprint(data)
		fmt.Println(hex.EncodeToString(sum[:]))
		return nil
	},
}

func init() {
	//rootCmd.AddCommand(fingerprintCmd)
}
