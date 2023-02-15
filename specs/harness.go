package specs

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen/backend"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

type SpecRelationships struct {
	Package string                `json:"package"`
	Preset  Preset                `json:"preset"`
	Defs    []ForkTypeDefinitions `json:"defs"`
}

type ForkTypeDefinitions struct {
	Fork  Fork           `json:"fork"`
	Types []TypeRelation `json:"types"`
}

type TypeRelation struct {
	SpecName string `json:"name"`
	TypeName string `json:"type_name"`
}

func ParseConfigFile(path string) (*SpecRelationships, error) {
	cb, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sr := &SpecRelationships{}
	err = yaml.Unmarshal(cb, sr)
	return sr, err
}

func (sr SpecRelationships) GoTypes() []string {
	tm := make(map[string]bool)
	gt := make([]string, 0)
	for _, d := range sr.Defs {
		for _, t := range d.Types {
			n := t.TypeName
			if n == "" {
				n = t.SpecName
			}
			if tm[n] {
				continue
			}
			gt = append(gt, n)
		}
	}
	return gt
}

func (sr SpecRelationships) RelationsAtFork(f Fork) (map[string]string, error) {
	// look up the position of this fork in the ordering of all forks, so we can start walking backwards from there
	fidx, err := forkIndex(f)
	if err != nil {
		return nil, err
	}
	// convert the list into a map for easier lookup
	fm := make(map[Fork][]TypeRelation)
	for _, d := range sr.Defs {
		fm[d.Fork] = d.Types
	}
	rf := make(map[string]string)
	// walk backwards through the forks to find the highest schema <= the requested fork
	for i := fidx; i >= 0; i-- {
		// get the fork for the current index
		f = ForkOrder[i]
		// get the list of type definitions at the given fork
		types, ok := fm[f]
		if !ok {
			// skip this fork if there are no type definitions for it in the given package
			continue
		}
		for _, t := range types {
			_, ok := rf[t.SpecName]
			// don't replace a newer version of the type with an older version
			if !ok {
				rf[t.SpecName] = t.TypeName
				// a blank Name proprety means the type name is the same as the spec name
				if rf[t.SpecName] == "" {
					rf[t.SpecName] = t.SpecName
				}
			}
		}
	}
	return rf, nil
}

type TestCaseTpl struct {
	ident      TestIdent
	fixture    Fixture
	structName string
}

func (tpl *TestCaseTpl) FixtureDirectory() string {
	return path.Join("testdata", tpl.fixture.Directory)
}

func (tpl *TestCaseTpl) rootPath() string {
	return path.Join(tpl.FixtureDirectory(), rootFilename)
}

func (tpl *TestCaseTpl) yamlPath() string {
	return path.Join(tpl.FixtureDirectory(), valueFilename)
}

func (tpl *TestCaseTpl) serializedPath() string {
	return path.Join(tpl.FixtureDirectory(), serializedFilename)
}

func (tpl *TestCaseTpl) ensureFixtures(fs afero.Fs) error {
	f := tpl.fixture
	if err := fs.MkdirAll(tpl.FixtureDirectory(), os.ModePerm); err != nil {
		return errors.Wrapf(err, "failed to create fixture directory %s", f.Directory)
	}
	if err := ensure(fs, tpl.rootPath(), f.Root.Contents, f.Root.FileMode); err != nil {
		return err
	}
	if err := ensure(fs, tpl.serializedPath(), f.Serialized.Contents, f.Serialized.FileMode); err != nil {
		return err
	}
	if err := ensure(fs, tpl.yamlPath(), f.Yaml.Contents, f.Yaml.FileMode); err != nil {
		return err
	}
	return nil
}

func ensure(fs afero.Fs, path string, contents []byte, mode os.FileMode) error {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return errors.Wrapf(err, "error checking for existence of %s", path)
	}
	if exists {
		return nil
	}
	if err := afero.WriteFile(fs, path, contents, mode); err != nil {
		return errors.Wrapf(err, "error writing fixture contents to %s", path)
	}
	return nil
}

func (tpl *TestCaseTpl) TestFuncName() string {
	id := tpl.ident
	return fmt.Sprintf("Test_%s_%s_%s_%d", id.Preset, id.Fork, tpl.structName, id.Offset)
}

func (tpl *TestCaseTpl) GoTypeName() string {
	return tpl.structName
}

func (tpl *TestCaseTpl) Render() (string, error) {
	b := bytes.NewBuffer(nil)
	err := testFuncBodyTpl.Execute(b, tpl)
	return b.String(), err
}

func WriteSpecTestFiles(cases map[TestIdent]Fixture, rels *SpecRelationships, fs afero.Fs) error {
	caseFuncs := make([]string, 0)
	fg := GroupByFork(cases)
	for _, fork := range ForkOrder {
		ids := fg[fork]
		raf, err := rels.RelationsAtFork(fork)
		if err != nil {
			return err
		}
		for _, id := range ids {
			structName, ok := raf[id.Name]
			if !ok {
				log.Infof("No implementation for %s, skipping test", cases[id].Directory)
				continue
			}
			tpl := &TestCaseTpl{
				ident:      id,
				fixture:    cases[id],
				structName: structName,
			}
			if err := tpl.ensureFixtures(fs); err != nil {
				return err
			}
			cfunc, err := tpl.Render()
			if err != nil {
				return err
			}
			caseFuncs = append(caseFuncs, cfunc)
		}
	}
	packageDecl := "package " + backend.RenderedPackageName(rels.Package) + "\n\n"
	contents := packageDecl + "\n\n" + testCaseTemplateImports + "\n\n" + strings.Join(caseFuncs, "\n\n")

	testBytes, err := format.Source([]byte(contents))
	if err != nil {
		return err
	}

	fname := "methodical_test.go"
	if err := afero.WriteFile(fs, fname, testBytes, 0666); err != nil {
		return errors.Wrapf(err, "error writing spectest functions to %s", fname)
	}

	return nil
}
