package cmd

import (
	"github.com/spf13/cobra"
)

var (
	receiveAddress string
	receiveFiles   []string
	receiveNames   []string
	receiveCmd     = &cobra.Command{
		Use:   "receive",
		Short: "asddzddefw",
		Long:  `asdzddfew`,
		Run: func(cmd *cobra.Command, args []string) {
			if receiveAddress != "" {
				//TODO the receiver sends a request to the address that is waiting for someone that ask for that file
			} else {
				//TODO the receiver wait to receive the file from someone in the correct port
			}
		},
	}
)

func init() {
	receiveCmd.Flags().StringVarP(&receiveAddress, "address", "a", "", "asd")
	receiveCmd.Flags().StringArrayVarP(&receiveFiles, "file", "f", []string{}, "asd")
	receiveCmd.Flags().StringArrayVarP(&receiveNames, "endpoint", "e", []string{}, "asd")
	rootCmd.AddCommand(receiveCmd)
}
