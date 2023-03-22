package sszgen

import (
	"fmt"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen/backend"
	"golang.org/x/tools/go/packages"
)

type GoPathScoper struct {
	packagePath string
	fieldNames  []string
	pkg         *types.Package
}

func NewGoPathScoper(packageName string) (*GoPathScoper, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports,
	}
	pkgs, err := packages.Load(cfg, []string{packageName}...)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if pkg.ID != packageName {
			continue
		}

		pp := &GoPathScoper{packagePath: pkg.ID, pkg: pkg.Types}
		return pp, nil
	}
	return nil, fmt.Errorf("package named '%s' could not be loaded from the go build system. Please make sure the current folder contains the go.mod for the target package, or that its go.mod is in a parent directory", packageName)
}

type PathScoper interface {
	Path() string
	Scope() *types.Scope
}

func (pp *GoPathScoper) Path() string {
	return pp.packagePath
}

func (pp *GoPathScoper) Scope() *types.Scope {
	return pp.pkg.Scope()
}

func TypeDefs(ps PathScoper, fieldNames ...string) ([]*TypeDef, error) {
	fileSet := token.NewFileSet()
	imp := importer.Default()

	// If no field names are requested, use all
	if fieldNames == nil {
		fieldNames = ps.Scope().Names()
	}

	results := make([]*TypeDef, len(fieldNames))
	for i, typeName := range fieldNames {
		typ, obj, err := lookupType(ps.Scope(), typeName)
		if err != nil {
			return nil, err
		}
		var mtyp *TypeDef
		if _, ok := typ.Underlying().(*types.Struct); ok {
			mtyp = newStructDef(fileSet, imp, typ, ps.Path())
		} else {
			mtyp = newPrimitiveDef(fileSet, imp, typ, ps.Path())
		}
		mtyp.object = obj
		results[i] = mtyp
	}
	return results, nil
}

var structTagRe = regexp.MustCompile(`\s+"(.*)"$`)

func reformatStructTag(line string) string {
	line = structTagRe.ReplaceAllString(line, " `$1`")
	return strings.ReplaceAll(line, `\"`, `"`)
}

func (pp *GoPathScoper) TypeDefSourceCode(defs []*TypeDef) ([]byte, error) {
	in := backend.NewImportNamer(pp.pkg.Path(), nil)
	structs := make([]string, 0)
	for _, def := range defs {
		obj := def.object
		defstring := types.ObjectString(obj, in.Name)
		// add a little whitespace for nicer formatting
		defstring = strings.ReplaceAll(defstring, ";", "\n")
		defstring = strings.ReplaceAll(defstring, "{", "{\n")
		defstring = strings.ReplaceAll(defstring, "}", "\n}\n")
		lines := strings.Split(defstring, "\n")
		for i := 0; i < len(lines); i++ {
			lines[i] = strings.TrimSpace(lines[i])
			lines[i] = reformatStructTag(lines[i])
		}
		structs = append(structs, strings.Join(lines, "\n"))
	}

	source := "package " + backend.RenderedPackageName(pp.pkg.Path()) + "\n\n" +
		in.ImportSource() +
		strings.Join(structs, "\n")
	return format.Source([]byte(source))
}
