package specs

import (
	"archive/tar"
	"cmp"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type Fixture struct {
	Directory      string
	RootFile       FixtureFile
	SerializedFile FixtureFile
	YamlFile       FixtureFile
	Ident          TestIdent
}

func (f *Fixture) Root() ([32]byte, error) {
	rc, err := f.RootFile.Contents()
	if err != nil {
		return [32]byte{}, err
	}
	return DecodeRootFile(rc)
}

func (f *Fixture) Serialized() ([]byte, error) {
	snappySer, err := f.SerializedFile.Contents()
	if err != nil {
		return nil, err
	}

	return snappy.Decode(nil, snappySer)
}

type FixtureFile interface {
	Contents() ([]byte, error)
	Mode() os.FileMode
}

type FixtureFileInMem struct {
	FileBytes []byte
	fileMode  os.FileMode
}

// NewFixtureFileInMem builds an in-memory fixture file with an explicit mode.
func NewFixtureFileInMem(b []byte, mode os.FileMode) *FixtureFileInMem {
	return &FixtureFileInMem{FileBytes: b, fileMode: mode}
}

type FixtureFs struct {
	fs   fs.FS
	path string
	mode os.FileMode
}

func (f *FixtureFileInMem) Contents() ([]byte, error) {
	return f.FileBytes, nil
}

func (f *FixtureFileInMem) Mode() os.FileMode {
	return f.fileMode
}

func (f *FixtureFs) Contents() (b []byte, err error) {
	fh, err := f.fs.Open(f.path)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := fh.Close()
		if err == nil {
			err = cerr
		}
	}()
	return io.ReadAll(fh)
}

func (f *FixtureFs) Mode() os.FileMode {
	return f.mode
}

func (f *Fixture) writeRoot(fs afero.Fs) error {
	c, err := f.RootFile.Contents()
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, path.Join(f.Directory, RootFilename), c, f.RootFile.Mode())
}

var (
	RootFilename       = "roots.yaml"
	SerializedFilename = "serialized.ssz_snappy"
	ValueFilename      = "value.yaml"
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

func ExtractTarballCases(tgz io.Reader, filter TestIdent) (map[TestIdent]Fixture, error) {
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
			c = Fixture{Directory: path.Dir(header.Name), Ident: ident}
		}
		f, err := io.ReadAll(tr)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading %s from spectest tarball", header.Name)
		}
		ff := &FixtureFileInMem{FileBytes: f, fileMode: os.FileMode(header.Mode)}
		switch fname {
		case RootFilename:
			c.RootFile = ff
		case SerializedFilename:
			c.SerializedFile = ff
		case ValueFilename:
			c.YamlFile = ff
		}
		cases[ident] = c
	}
	return cases, nil
}

func ExtractFsCases(f fs.FS, filter TestIdent) (map[TestIdent]Fixture, error) {
	cases := make(map[TestIdent]Fixture)
	return cases, fs.WalkDir(f, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ident, fname, err := ParsePath(fpath)
		if err != nil {
			return err
		}
		if !filter.Match(ident) {
			return nil
		}
		c, ok := cases[ident]
		if !ok {
			c = Fixture{Directory: path.Dir(fpath), Ident: ident}
		}
		ff := &FixtureFs{fs: f, path: fpath, mode: d.Type()}
		switch fname {
		case RootFilename:
			c.RootFile = ff
		case SerializedFilename:
			c.SerializedFile = ff
		case ValueFilename:
			c.YamlFile = ff
		}
		cases[ident] = c
		return nil
	})
}

func NewGroupedTestCases(cases map[TestIdent]Fixture) GroupedTestCases {
	grouped := GroupCasesByType(cases)
	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	slices.Sort(names)

	return GroupedTestCases{Types: names, testCases: grouped}
}

type GroupedTestCases struct {
	Types     []string
	testCases map[string][]Fixture
}

func (g GroupedTestCases) FixturesForType(name string) []Fixture {
	cases := g.testCases[name]
	slices.SortFunc(cases, func(a, b Fixture) int {
		return cmp.Compare(a.Ident.Offset, b.Ident.Offset)
	})
	return cases
}

func GroupCasesByType(cases map[TestIdent]Fixture) map[string][]Fixture {
	grouped := make(map[string][]Fixture)
	for ident, fixture := range cases {
		grouped[ident.Name] = append(grouped[ident.Name], fixture)
	}
	return grouped
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
	rpath := path.Join(dir, RootFilename)
	rootBytes, err := os.ReadFile(rpath)
	if err != nil {
		return [32]byte{}, nil, errors.Wrapf(err, "error reading expected root fixture file %s", rpath)
	}
	root, err := DecodeRootFile(rootBytes)
	if err != nil {
		return [32]byte{}, nil, errors.Wrapf(err, "error decoding expected root fixture file %s, hex contents=%#x", rpath, rootBytes)
	}

	spath := path.Join(dir, SerializedFilename)
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
