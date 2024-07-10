package sszgen

import (
	"go/format"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen/backend"
	"golang.org/x/tools/go/packages"
)

type GoPathScoper struct {
	packagePath string
	pkg         *packages.Package
	imp         *Importer
}

func NewGoPathScoper(packageName string) (*GoPathScoper, error) {
	imp := NewImporter()
	pkg, err := imp.Load(packageName)
	if err != nil {
		return nil, err
	}
	return &GoPathScoper{pkg: pkg, imp: imp}, nil
}

type PathScoper interface {
	Path() string
	Scope() *types.Scope
	Importer() *Importer
}

func (pp *GoPathScoper) Path() string {
	return pp.pkg.PkgPath
}

func (pp *GoPathScoper) Scope() *types.Scope {
	return pp.pkg.Types.Scope()
}

func (pp *GoPathScoper) Importer() *Importer {
	return pp.imp
}

func TypeDefs(ps PathScoper, fieldNames ...string) ([]*TypeDef, error) {
	fileSet := token.NewFileSet()

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
			mtyp = newStructDef(fileSet, ps.Importer(), typ, ps.Path())
		} else {
			mtyp = newPrimitiveDef(fileSet, ps.Importer(), typ, ps.Path())
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
	in := backend.NewImportNamer(pp.Path(), nil)
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

	source := "package " + backend.RenderedPackageName(pp.Path()) + "\n\n" +
		in.ImportSource() +
		strings.Join(structs, "\n")
	return format.Source([]byte(source))
}
