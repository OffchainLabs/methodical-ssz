package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen"
	"github.com/OffchainLabs/methodical-ssz/sszgen/config"
	"github.com/OffchainLabs/methodical-ssz/sszgen/render"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/urfave/cli/v2"
)

var sourcePackage, typeNamesArg, genConfigPath, output, packageNameOverride string
var disableDelegation, disableProgressive bool

var packageFlag = &cli.StringFlag{
	Name:        "package",
	Value:       "",
	Destination: &sourcePackage,
}

var typesFlag = &cli.StringFlag{
	Name:        "type-names",
	Value:       "",
	Usage:       "if specified, only generate methods for types specified in this comma-separated list.",
	Destination: &typeNamesArg,
}

var genConfigFlag = &cli.StringFlag{
	Name:        "config",
	Value:       "",
	Usage:       "path to a yaml file containing codegen configuration.",
	Destination: &genConfigPath,
}

var Generate = &cli.Command{
	Name:      "generate",
	ArgsUsage: "<input package, eg github.com/prysmaticlabs/prysm/v3/proto/beacon/p2p/v1>",
	Aliases:   []string{"gen"},
	Usage:     "generate methodsets for a go struct type to support ssz ser/des",
	Flags: []cli.Flag{
		packageFlag,
		typesFlag,
		genConfigFlag,
		&cli.StringFlag{
			Name:        "output",
			Value:       "",
			Usage:       "directory to write generated code (same as input by default).",
			Destination: &output,
		},
		&cli.BoolFlag{
			Name:        "disable-delegation",
			Usage:       "If specified, do not check for existing ssz method sets. helpful when the codegen source has them already.",
			Destination: &disableDelegation,
		},
		&cli.BoolFlag{
			Name:        "disable-progressive",
			Usage:       "If specified, ignore all progressive config: containers merkleize as standard SSZ containers and progressive collections (list, byte list, bitlist) downgrade to their standard bounded forms.",
			Destination: &disableProgressive,
		},
		&cli.StringFlag{
			Name:        "override-package-name",
			Value:       "",
			Usage:       "Override the default package name (last component of package import path).",
			Destination: &packageNameOverride,
		},
	},
	Action: func(c *cli.Context) error {
		gc, err := genConfig(sourcePackage, typeNamesArg, genConfigPath)
		if err != nil {
			return err
		}
		if disableProgressive {
			gc.DisableProgressive()
		}
		fmt.Printf("Parsing package %v\n", gc.Package)
		ps, err := sszgen.NewGoPathScoper(gc.Package, gc)
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

		defs, err := sszgen.TypeDefs(ps, gc.TypeNames()...)
		if err != nil {
			return err
		}
		opts := make([]sszgen.FieldParserOpt, 0)
		if disableDelegation {
			opts = append(opts, sszgen.WithDisableDelegation())
		}
		vrs := make([]gentypes.ValRep, 0, len(defs))
		for _, s := range defs {
			fmt.Printf("Generating methods for %s/%s\n", s.PackageName, s.Name)
			typeRep, err := sszgen.ParseTypeDef(s, opts...)
			if err != nil {
				return err
			}
			vrs = append(vrs, typeRep)
		}
		fmt.Println("Rendering template")
		rbytes, err := render.Render(gc.Package, packageNameOverride, vrs)
		if err != nil {
			return err
		}
		_, err = io.Copy(outFh, bytes.NewReader(rbytes))
		return err
	},
}

func genConfig(pkgArg, typesArg, genspecPath string) (*config.GeneratorConfig, error) {
	if genspecPath != "" {
		if pkgArg != "" || typesArg != "" {
			return nil, fmt.Errorf("cannot specify both %s and %s/%s flags", genConfigFlag.Name, packageFlag.Name, typesFlag.Name)
		}
		gsb, err := os.ReadFile(genspecPath)
		if err != nil {
			return nil, err
		}
		return config.ParseGeneratorConfig(gsb)
	}
	if pkgArg == "" {
		return nil, fmt.Errorf("missing required %s flag", packageFlag.Name)
	}
	if typesArg == "" {
		return nil, fmt.Errorf("missing required %s flag", typesFlag.Name)
	}
	fields := strings.Split(strings.TrimSpace(typesArg), ",")
	return config.NewGeneratorConfig(pkgArg, fields)
}
