package specs

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type TestCase struct {
	Root       [32]byte
	Serialized []byte
	Yaml       []byte
}

var (
	rootFilename       = "roots.yaml"
	serializedFilename = "serialized.ssz_snappy"
	valueFilename      = "value.yaml"
)

func IdentFilter(ident TestIdent) func([]TestIdent) []TestIdent {
	return func(maybe []TestIdent) []TestIdent {
		matches := make([]TestIdent, 0)
		for _, m := range maybe {
			if ident.Match(m) {
				matches = append(matches, m)
			}
		}
		return matches
	}
}

func ExtractCases(tgz io.Reader, filter TestIdent) (map[TestIdent]TestCase, error) {
	cases := make(map[TestIdent]TestCase)
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
		ident, fname, err := ParsePath(header.Name)
		if err != nil {
			return nil, err
		}
		if !filter.Match(ident) {
			continue
		}
		c, ok := cases[ident]
		if !ok {
			c = TestCase{}
		}
		f, err := io.ReadAll(tr)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading %s from spectest tarball", header.Name)
		}
		switch fname {
		case rootFilename:
			r, err := decodeRootFile(f)
			if err != nil {
				return nil, errors.Wrapf(err, "error decoding contents of %s from spectest tarball", header.Name)
			}
			c.Root = r
		case serializedFilename:
			s, err := snappy.Decode(nil, f)
			if err != nil {
				return nil, errors.Wrapf(err, "err decoding %s from spectest tarball as snappy-encoding", header.Name)
			}
			c.Serialized = s
		case valueFilename:
			c.Yaml = f
		}
		cases[ident] = c
	}
	return cases, nil
}

func decodeRootFile(f []byte) ([32]byte, error) {
	root := [32]byte{}
	ry := &struct {
		Root string `json:"root"`
	}{}
	if err := yaml.Unmarshal(f, ry); err != nil {
		return root, err
	}
	br, err := hexutil.Decode(ry.Root)
	if err != nil {
		return root, err
	}
	copy(root[:], br)
	return root, nil
}
