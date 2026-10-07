package commands

import (
	"os"

	"github.com/OffchainLabs/methodical-ssz/specs"
	"github.com/OffchainLabs/methodical-ssz/sszgen/backend/spectest"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
)

var localPackageName string

// GenSpectest generates spec tests in local/consumer mode: run from inside the
// module that owns the types. It does no package loading or codegen; the
// generated test-only package exercises the ssz methodsets already committed there.
var GenSpectest = &cli.Command{
	Name:  "gen-spectest",
	Usage: "generate spec tests into the module that owns the types; run from that module's directory (see README)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "release-uri",
			Usage:       "url or file in file:// format pointing at a github.com/ethereum/consensus-spec-tests release",
			Required:    true,
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
			Usage:       "directory to write the generated test package and its testdata fixtures",
			Required:    true,
			Destination: &output,
		},
		&cli.StringFlag{
			Name:        "package-name",
			Usage:       "package name for the generated test file (default: " + spectest.DefaultLocalPackageName + ")",
			Destination: &localPackageName,
		},
	},
	Action: func(c *cli.Context) error {
		return actionGenSpectest(c)
	},
}

func actionGenSpectest(_ *cli.Context) error {
	if err := os.MkdirAll(output, os.ModePerm); err != nil {
		return errors.Wrapf(err, "failed to create output directory %s", output)
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), output)
	cfg, err := spectest.ParseConfigFile(configPath)
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
	return spectest.WriteLocalSpecTestFiles(cases, cfg, fs, localPackageName)
}
