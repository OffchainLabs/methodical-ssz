package main

import (
	"io"
	"os"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen"
	"github.com/OffchainLabs/methodical-ssz/sszgen/testutil"
	"github.com/urfave/cli/v2"
)

var ir = &cli.Command{
	Name:      "ir",
	ArgsUsage: "<input package, eg github.com/prysmaticlabs/prysm/v3/proto/beacon/p2p/v1>",
	Aliases:   []string{"gen"},
	Usage:     "generate intermediate representation for a go struct type. This data structure is used by the backend code generator. Outputting it to a source file an be useful for generating test cases and debugging.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Value:       "",
			Usage:       "file path to write generated code",
			Destination: &output,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "type-names",
			Value:       "",
			Usage:       "if specified, only generate types specified in this comma-separated list",
			Destination: &typeNames,
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() > 0 {
			sourcePackage = c.Args().Get(0)
		}

		var err error
		var fields []string
		if len(typeNames) > 0 {
			fields = strings.Split(strings.TrimSpace(typeNames), ",")
		}
		ps, err := sszgen.NewGoPathScoper(sourcePackage)
		if err != nil {
			return err
		}

		outFh, err := os.Create(output)
		if err != nil {
			return err
		}
		defer outFh.Close()

		renderedTypes := make([]string, 0)
		defs, err := sszgen.TypeDefs(ps, fields...)
		if err != nil {
			return err
		}
		for _, s := range defs {
			typeRep, err := sszgen.ParseTypeDef(s)
			if err != nil {
				return err
			}
			rendered, err := testutil.RenderIntermediate(typeRep)
			if err != nil {
				return err
			}
			renderedTypes = append(renderedTypes, rendered)
		}
		if err != nil {
			return err
		}

		_, err = io.Copy(outFh, strings.NewReader(strings.Join(renderedTypes, "\n")))
		return err
	},
}
