package specs

import (
	"archive/tar"
	"compress/gzip"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
)

type TestCase struct {
	Ident TestIdent
	Root [32]byte
	Serialized []byte
	Yaml []byte
}

func ExtractCases(tgz io.Reader, filter TestIdent) ([]TestCase, error) {
	cases := make([]TestCase, 0)
	uncompressed, err := gzip.NewReader(tgz)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read gzip-compressed stream")
	}
	tr := tar.NewReader(uncompressed)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file header from spectest tarball")
		}
		ident, err := ParsePath(header.Name)
		if err != nil {
			return nil, err
		}
		if !filter.Match(ident) {
			log.Infof("skipping path %s", header.Name)
			continue
		}
		cases = append(cases, TestCase{Ident: ident})
	}
	return cases, nil
}