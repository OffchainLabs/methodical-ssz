package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen"
	"github.com/OffchainLabs/methodical-ssz/sszgen/backend"
	"github.com/urfave/cli/v2"
)

var sourcePackage, output, typeNames string
var generate = &cli.Command{
	Name:      "generate",
	ArgsUsage: "<input package, eg github.com/prysmaticlabs/prysm/v3/proto/beacon/p2p/v1>",
	Aliases:   []string{"gen"},
	Usage:     "generate methodsets for a go struct type to support ssz ser/des",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Value:       "",
			Usage:       "directory to write generated code (same as input by default)",
			Destination: &output,
		},
		&cli.StringFlag{
			Name:        "type-names",
			Value:       "",
			Usage:       "if specified, only generate methods for types specified in this comma-separated list",
			Destination: &typeNames,
		},
	},
	Action: func(c *cli.Context) error {
		sourcePackage = c.Args().Get(0)
		if sourcePackage == "" {
			cli.ShowCommandHelp(c, "generate")
			return fmt.Errorf("error: mising required <input package> argument")
		}
		var err error
		var fields []string
		if len(typeNames) > 0 {
			fields = strings.Split(strings.TrimSpace(typeNames), ",")
		}

		fmt.Printf("Parsing package %v\n", sourcePackage)
		parser, err := sszgen.NewPackageParser(sourcePackage, fields)
		if err != nil {
			return err
		}

		if output == "" {
			output = "methodical.ssz.go"
		}
		outFh, err := os.Create(output)
		if err != nil {
			return err
		}
		defer outFh.Close()

		g := backend.NewGenerator(sourcePackage, sourcePackage)
		for _, s := range parser.TypeDefs() {
			fmt.Printf("Generating methods for %s/%s\n", s.PackageName, s.Name)
			typeRep, err := sszgen.ParseTypeDef(s)
			if err != nil {
				return err
			}
			g.Generate(typeRep)
		}
		fmt.Println("Rendering template")
		rbytes, err := g.Render()
		if err != nil {
			return err
		}
		_, err = io.Copy(outFh, bytes.NewReader(rbytes))
		return err
	},
}
