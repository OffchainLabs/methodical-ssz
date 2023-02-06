package main

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/kasey/methodical-ssz/specs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var releaseURI string
var tests = &cli.Command{
	Name:  "spectest",
	Usage: "generate go test methods to execute spectests against generated types",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "release-uri",
			Value:       "",
			Usage:       "url or file in file:// format pointing at a github.com/ethereum/consensus-spec-tests release",
			Destination: &releaseURI,
		},
	},
	Action: func(c *cli.Context) error {
		return actionSpectests(c)
	},
}

func actionSpectests(cl *cli.Context) error {
	log.Infof("releaseURI=%s", releaseURI)
	r, err := loadArchive(releaseURI)
	if err != nil {
		return err
	}
	cases, err := specs.ExtractCases(r, specs.TestIdent{Preset: specs.Mainnet})
	if err != nil {
		return err
	}
	for ident, _ := range cases {
		fmt.Printf("%s\n", ident)
	}
	return nil
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
