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

type ImportNamer struct {
	source *types.Package
	// imports picks aliases for colliding package names
	imports     map[*types.Package]string
	importNames map[string]*types.Package
}

func (n *ImportNamer) Name(p *types.Package) string {
	// no package name for self
	if p == n.source {
		return ""
	}
	name, exists := n.imports[p]
	if exists {
		return name
	}
	// build increasingly long path suffixes until a unique one is found
	parts := strings.Split(p.Path(), "/")
	for i := 0; i < len(parts); i++ {
		name := strings.Join(parts[len(parts)-1-i:], "_")
		// deal with domain portion of path for extreme case where 2 packages only differ in domain
		name = strings.ReplaceAll(name, ".", "_")
		// dashes are valid in package names but not go identifiers - like go-bitfield
		name = strings.ReplaceAll(name, "-", "_")
		_, conflict := n.importNames[name]
		if conflict {
			continue
		}
		n.importNames[name] = p
		n.imports[p] = name
		return name
	}
	panic(fmt.Sprintf("unable to find unique name for package %s", p.Path()))
}

func (n *ImportNamer) ImportSource() string {
	imports := make([]string, 0)
	for alias, pkg := range n.importNames {
		if pkg.Path() == "google.golang.org/protobuf/internal/impl" {
			imports = append(imports, fmt.Sprintf("%s \"google.golang.org/protobuf/runtime/protoimpl\"", alias))
		} else {
			imports = append(imports, fmt.Sprintf("%s \"%s\"", alias, pkg.Path()))
		}
	}
	return fmt.Sprintf("import (\n%s\n)\n", strings.Join(imports, "\n"))
}

func NewImportNamer(source *types.Package) *ImportNamer {
	return &ImportNamer{
		source:      source,
		imports:     make(map[*types.Package]string),
		importNames: make(map[string]*types.Package),
	}
}

var structTagRe = regexp.MustCompile(`\s+"(.*)"$`)

func reformatStructTag(line string) string {
	line = structTagRe.ReplaceAllString(line, " `$1`")
	return strings.ReplaceAll(line, `\"`, `"`)
}

func (pp *GoPathScoper) TypeDefSourceCode(defs []*TypeDef) ([]byte, error) {
	in := NewImportNamer(pp.pkg)
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
