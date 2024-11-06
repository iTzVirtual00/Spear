package cmd

import (
	"github.com/spf13/cobra"
)

var (
	sendAddress string
	sendFiles   []string
	sendNames   []string
	sendCmd     = &cobra.Command{
		Use:   "send",
		Short: "asdddefw",
		Long:  `asdddfew`,
		Run: func(cmd *cobra.Command, args []string) {
			if sendAddress != "" {
				//TODO the sender sends the file to the address
			} else {
				//TODO the sender waits for someone to establish a new connection to the correct port and send the file to that person
			}
		},
	}
)

func init() {
	sendCmd.Flags().StringVarP(&sendAddress, "address", "a", "", "asd")
	sendCmd.Flags().StringArrayVarP(&sendFiles, "file", "f", []string{}, "asd")
	sendCmd.Flags().StringArrayVarP(&sendNames, "endpoint", "e", []string{}, "asd")
	rootCmd.AddCommand(sendCmd)
}
