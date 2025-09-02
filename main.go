package main

import (
	"log"
	"spear/cli"
	"spear/config"
)

func main() {

	spearConfig, err := config.LoadConfig("spear.yml")

	if err != nil {
		log.Fatal(err)
		return
	}
	cli.RunCLI(*spearConfig) // actual code in ./handlers
}
