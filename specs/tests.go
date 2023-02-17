package specs

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Fixture struct {
	Directory  string
	Root       FixtureFile
	Serialized FixtureFile
	Yaml       FixtureFile
}

type FixtureFile struct {
	Contents []byte
	FileMode os.FileMode
}

func (f *Fixture) writeRoot(fs afero.Fs) error {
	return afero.WriteFile(fs, path.Join(f.Directory, rootFilename), f.Root.Contents, f.Root.FileMode)
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

func GroupByFork(cases map[TestIdent]Fixture) map[Fork][]TestIdent {
	m := make(map[Fork][]TestIdent)
	for id := range cases {
		switch len(m[id.Fork]) {
		case 0:
			m[id.Fork] = []TestIdent{id}
		case 1:
			m[id.Fork] = append(m[id.Fork], id)
		default:
			for i, cur := range m[id.Fork] {
				if id.LessThan(cur) {
					m[id.Fork] = append(m[id.Fork][:i], append([]TestIdent{id}, m[id.Fork][i:]...)...)
					break
				}
			}
		}
	}
	return m
}

func GroupByType(ti []TestIdent) map[string][]TestIdent {
	m := make(map[string][]TestIdent)
	for _, t := range ti {
		m[t.Name] = append(m[t.Name], t)
	}
	return m
}

func ExtractCases(tgz io.Reader, filter TestIdent) (map[TestIdent]Fixture, error) {
	cases := make(map[TestIdent]Fixture)
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
			c = Fixture{Directory: path.Dir(header.Name)}
		}
		f, err := io.ReadAll(tr)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading %s from spectest tarball", header.Name)
		}
		switch fname {
		case rootFilename:
			c.Root = FixtureFile{Contents: f, FileMode: os.FileMode(header.Mode)}
		case serializedFilename:
			c.Serialized = FixtureFile{Contents: f, FileMode: os.FileMode(header.Mode)}
		case valueFilename:
			c.Yaml = FixtureFile{Contents: f, FileMode: os.FileMode(header.Mode)}
		}
		cases[ident] = c
	}
	return cases, nil
}

func DecodeRootFile(f []byte) ([32]byte, error) {
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

func RootAndSerializedFromFixture(dir string) ([32]byte, []byte, error) {
	rpath := path.Join(dir, rootFilename)
	rootBytes, err := os.ReadFile(rpath)
	if err != nil {
		return [32]byte{}, nil, errors.Wrapf(err, "error reading expected root fixture file %s", rpath)
	}
	root, err := DecodeRootFile(rootBytes)
	if err != nil {
		return [32]byte{}, nil, errors.Wrapf(err, "error decoding expected root fixture file %s, hex contents=%#x", rpath, rootBytes)
	}

	spath := path.Join(dir, serializedFilename)
	snappySer, err := os.ReadFile(spath)
	if err != nil {
		return [32]byte{}, nil, errors.Wrapf(err, "error reading serialized fixture file %s", spath)
	}
	serialized, err := snappy.Decode(nil, snappySer)
	if err != nil {
		return [32]byte{}, nil, errors.Wrapf(err, "error snappy decoding serialized fixture file %s", spath)
	}

	return root, serialized, nil
}
