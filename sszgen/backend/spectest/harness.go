package spectest

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/specs"
	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

type SpecRelationships struct {
	Package string                `json:"package"`
	Preset  specs.Preset          `json:"preset"`
	Defs    []ForkTypeDefinitions `json:"defs"`
}

type ForkTypeDefinitions struct {
	Fork  specs.Fork     `json:"fork"`
	Types []TypeRelation `json:"types"`
}

type TypeRelation struct {
	SpecName string `json:"name"`
	TypeName string `json:"type_name"`
	// Package optionally names the Go package declaring this type, for
	// consumers whose types span multiple packages (gen-spectest local mode).
	// Blank means the type lives in the config's top-level package.
	Package string `json:"package"`
}

// ResolvedType is a spec type resolved to its Go type name and declaring
// package. The package is always concrete (a blank override resolves to the
// config's top-level package).
type ResolvedType struct {
	TypeName string
	Package  string
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

func (sr SpecRelationships) RelationsAtFork(f specs.Fork) (map[string]ResolvedType, error) {
	// look up the position of this fork so we can start walking backwards from there
	fidx, err := specs.ForkIndex(f)
	if err != nil {
		return nil, err
	}
	fm := make(map[specs.Fork][]TypeRelation)
	for _, d := range sr.Defs {
		fm[d.Fork] = d.Types
	}
	rf := make(map[string]ResolvedType)
	// walk backwards through the forks to find the highest schema <= the requested fork
	for i := fidx; i >= 0; i-- {
		f = specs.ForkOrder[i]
		types, ok := fm[f]
		if !ok {
			continue
		}
		for _, t := range types {
			// keep the newest definition (first seen, walking backward); it
			// overrides both name and package.
			if _, ok := rf[t.SpecName]; ok {
				continue
			}
			rt := ResolvedType{TypeName: t.TypeName, Package: t.Package}
			// a blank type_name means the type name is the same as the spec name
			if rt.TypeName == "" {
				rt.TypeName = t.SpecName
			}
			// a blank package means the type lives in the top-level package
			if rt.Package == "" {
				rt.Package = sr.Package
			}
			rf[t.SpecName] = rt
		}
	}
	return rf, nil
}

type TestCaseTpl struct {
	ident      specs.TestIdent
	fixture    specs.Fixture
	structName string
	// qualifier is the package qualifier for type references; empty when the
	// test file lives in the same package as the types (the standalone
	// spectest mode), the types package's name in local/consumer mode.
	qualifier string
}

func (tpl *TestCaseTpl) FixtureDirectory() string {
	return path.Join("testdata", tpl.fixture.Directory)
}

func (tpl *TestCaseTpl) rootPath() string {
	return path.Join(tpl.FixtureDirectory(), specs.RootFilename)
}

func (tpl *TestCaseTpl) yamlPath() string {
	return path.Join(tpl.FixtureDirectory(), specs.ValueFilename)
}

func (tpl *TestCaseTpl) serializedPath() string {
	return path.Join(tpl.FixtureDirectory(), specs.SerializedFilename)
}

func (tpl *TestCaseTpl) ensureFixtures(fs afero.Fs) error {
	f := tpl.fixture
	if err := fs.MkdirAll(tpl.FixtureDirectory(), os.ModePerm); err != nil {
		return errors.Wrapf(err, "failed to create fixture directory %s", f.Directory)
	}
	rc, err := f.RootFile.Contents()
	if err != nil {
		return err
	}
	if err := ensure(fs, tpl.rootPath(), rc, f.RootFile.Mode()); err != nil {
		return err
	}

	sc, err := f.SerializedFile.Contents()
	if err != nil {
		return err
	}
	if err := ensure(fs, tpl.serializedPath(), sc, f.SerializedFile.Mode()); err != nil {
		return err
	}

	yc, err := f.YamlFile.Contents()
	if err != nil {
		return err
	}
	if err := ensure(fs, tpl.yamlPath(), yc, f.YamlFile.Mode()); err != nil {
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
	if tpl.qualifier != "" {
		return tpl.qualifier + "." + tpl.structName
	}
	return tpl.structName
}

func (tpl *TestCaseTpl) Render() (string, error) {
	b := bytes.NewBuffer(nil)
	err := testFuncBodyTpl.Execute(b, tpl)
	return b.String(), err
}

// renderCaseFuncs walks the cases in canonical fork order, resolves each spec
// type to its go type via the config's fork-inheritance rules, materializes
// the fixtures onto fs, and renders the test functions. aliases maps a
// resolved type's package to the import alias qualifying its references; a
// nil map leaves references unqualified (test files living in the types
// package, the standalone mode).
func renderCaseFuncs(cases map[specs.TestIdent]specs.Fixture, rels *SpecRelationships, fs afero.Fs, aliases map[string]string) ([]string, error) {
	caseFuncs := make([]string, 0)
	fg := specs.GroupByFork(cases)
	for _, fork := range specs.ForkOrder {
		ids := fg[fork]
		raf, err := rels.RelationsAtFork(fork)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			rt, ok := raf[id.Name]
			if !ok {
				log.Infof("No implementation for %s, skipping test", cases[id].Directory)
				continue
			}
			tpl := &TestCaseTpl{
				ident:      id,
				fixture:    cases[id],
				structName: rt.TypeName,
				qualifier:  aliases[rt.Package],
			}
			if err := tpl.ensureFixtures(fs); err != nil {
				return nil, err
			}
			cfunc, err := tpl.Render()
			if err != nil {
				return nil, err
			}
			caseFuncs = append(caseFuncs, cfunc)
		}
	}
	return caseFuncs, nil
}

func WriteSpecTestFiles(cases map[specs.TestIdent]specs.Fixture, rels *SpecRelationships, fs afero.Fs) error {
	caseFuncs, err := renderCaseFuncs(cases, rels, fs, nil)
	if err != nil {
		return err
	}
	packageDecl := "package " + core.RenderedPackageName(rels.Package) + "\n\n"
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
