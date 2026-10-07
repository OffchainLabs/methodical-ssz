package commands

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/OffchainLabs/methodical-ssz/specs"
	"github.com/OffchainLabs/methodical-ssz/sszgen"
	"github.com/OffchainLabs/methodical-ssz/sszgen/backend/spectest"
	"github.com/OffchainLabs/methodical-ssz/sszgen/render"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
)

var releaseURI, configPath string
var Tests = &cli.Command{
	Name:  "spectest",
	Usage: "generate go test methods to execute spectests against generated types",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "release-uri",
			Usage:       "url or file in file:// format pointing at a github.com/ethereum/consensus-spec-tests release",
			Destination: &releaseURI,
		},
		&cli.StringFlag{
			Name:        "config",
			Usage:       "path to yaml file configuring spec test relationships, see readme or prysm example for format",
			Required:    true,
			Destination: &configPath,
		},
		&cli.StringFlag{
			Name:        "output",
			Usage:       "path to output directory where spec test package and copy of consensus types and ssz methods will be written",
			Required:    true,
			Destination: &output,
		},
		&cli.BoolFlag{
			Name:        "disable-delegation",
			Usage:       "if specified, do not check for existing ssz method sets. helpful when the codegen source has them already",
			Destination: &disableDelegation,
		},
	},
	Action: func(c *cli.Context) error {
		return actionSpectests(c)
	},
}

func actionSpectests(_ *cli.Context) error {
	if err := os.MkdirAll(output, os.ModePerm); err != nil {
		return errors.Wrapf(err, "failed to create output directory %s", output)
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), output)
	cfg, err := spectest.ParseConfigFile(configPath)
	if err != nil {
		return err
	}
	types := cfg.GoTypes()
	ps, err := sszgen.NewGoPathScoper(cfg.Package, nil)
	if err != nil {
		return err
	}

	r, err := loadArchive(releaseURI)
	if err != nil {
		return errors.Wrapf(err, "failed to open spectest archive from uri %s", releaseURI)
	}
	cases, err := specs.ExtractTarballCases(r, specs.TestIdent{Preset: specs.Mainnet})
	if err != nil {
		return err
	}
	for ident := range cases {
		fmt.Printf("%s\n", ident)
	}

	defs, err := sszgen.TypeDefs(ps, types...)
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
	rbytes, err := render.Render(cfg.Package, "", vrs)
	if err != nil {
		return err
	}
	if err := afero.WriteFile(fs, "methodical.ssz.go", rbytes, 0666); err != nil {
		return err
	}
	source, err := ps.TypeDefSourceCode(defs)
	if err != nil {
		return err
	}
	if err := afero.WriteFile(fs, "structs.go", source, 0666); err != nil {
		return err
	}
	return spectest.WriteSpecTestFiles(cases, cfg, fs)
}

func loadArchive(uri string) (io.Reader, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "file" {
		return os.Open(u.Path)
	}
	return nil, errors.New("unsupported url protocol")
}
