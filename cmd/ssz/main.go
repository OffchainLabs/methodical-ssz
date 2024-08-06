package main

import (
	"log"
	"os"

	"github.com/OffchainLabs/methodical-ssz/cmd/ssz/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Usage:    "ssz codegen tools",
		Commands: commands.All,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
