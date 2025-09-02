package cmd

import (
	"net"
	"spear/config"
	"spear/utils"

	"github.com/spf13/cobra"
)

var (
	DefaultListenValue = utils.TCPAddr{Addr: &net.TCPAddr{IP: net.IPv4zero, Port: utils.SpearDefaultPort}}
	spearConfig        config.SpearConfig
	rootCmd            = &cobra.Command{
		Use:   "spear",
		Short: "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	}
)

func Execute(config config.SpearConfig) error {
	spearConfig = config
	return rootCmd.Execute()
}
