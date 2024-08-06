package sszgen

import (
	"fmt"
	"go/format"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/sszgen/config"
	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

var errNamedTypeUnresolvable = errors.New("unresolvable named type")

type PathScoper interface {
	Path() string
	Scope() *types.Scope
	TypeConfig(string) config.TypeConfig
}

type GoPathScoper struct {
	packagePath string
	pkg         *packages.Package
	gcfg        *config.GeneratorConfig
}

func NewGoPathScoper(packageName string, gcfg *config.GeneratorConfig) (*GoPathScoper, error) {
	pkg, err := NewImporter().Load(packageName)
	if err != nil {
		return nil, err
	}
	return &GoPathScoper{pkg: pkg, gcfg: gcfg}, nil
}

func (pp *GoPathScoper) Path() string {
	return pp.pkg.PkgPath
}

func (pp *GoPathScoper) Scope() *types.Scope {
	return pp.pkg.Types.Scope()
}

func (pp *GoPathScoper) TypeConfig(name string) config.TypeConfig {
	if pp.gcfg == nil {
		return config.TypeConfig{}
	}
	return pp.gcfg.TypeConfig(name)
}

func TypeDefs(ps PathScoper, fieldNames ...string) ([]*TypeDef, error) {
	fileSet := token.NewFileSet()

	if fieldNames == nil {
		fieldNames = ps.Scope().Names()
	}

	results := make([]*TypeDef, len(fieldNames))
	for i, typeName := range fieldNames {
		typ, err := resolveTypeDef(ps, typeName, fileSet)
		if err != nil {
			return nil, err
		}
		results[i] = typ
	}
	if err := validateConfigs(results); err != nil {
		return nil, err
	}
	return results, nil
}

var structTagRe = regexp.MustCompile(`\s+"(.*)"$`)

func reformatStructTag(line string) string {
	line = structTagRe.ReplaceAllString(line, " `$1`")
	return strings.ReplaceAll(line, `\"`, `"`)
}

func (pp *GoPathScoper) TypeDefSourceCode(defs []*TypeDef) ([]byte, error) {
	in := core.NewImportNamer(pp.Path(), nil)
	structs := make([]string, 0)
	for _, def := range defs {
		obj := def.object
		defstring := types.ObjectString(obj, in.Name)
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

	source := "package " + core.RenderedPackageName(pp.Path()) + "\n\n" +
		in.ImportSource() +
		strings.Join(structs, "\n")
	return format.Source([]byte(source))
}

func resolveTypeDef(pp PathScoper, name string, fileSet *token.FileSet) (*TypeDef, error) {
	obj := pp.Scope().Lookup(name)
	typ, err := resolveNamedType(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "name=%s", name)
	}
	if isStructType(obj) {
		return newStructDef(fileSet, pp, typ, obj), nil
	}
	return newPrimitiveDef(fileSet, pp, typ, obj), nil
}

func resolveNamedType(obj types.Object) (*types.Named, error) {
	if obj == nil {
		return nil, errors.Wrap(errNamedTypeUnresolvable, "nil lookup")
	}
	if typ, ok := obj.(*types.TypeName); ok {
		return typ.Type().(*types.Named), nil
	}
	msg := fmt.Sprintf("could not resolve symbol %s from package %s", obj.String(), obj.Pkg().Path())
	return nil, errors.Wrap(errNamedTypeUnresolvable, msg)
}

func isStructType(obj types.Object) bool {
	_, ok := obj.Type().Underlying().(*types.Struct)
	return ok
}
