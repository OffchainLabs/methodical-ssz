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

type packageParser struct {
	// configuration
	packagePath string
	fieldNames  []string
	// parsed values
	pkg     *types.Package
	results []*TypeDef
	// keeping the object form allows the source code for type definitions to be regenerated for spectests
	objects map[string]types.Object
}

func NewPackageParser(packageName string, fieldNames []string) (*packageParser, error) {
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

		pp := &packageParser{packagePath: pkg.ID, fieldNames: fieldNames, pkg: pkg.Types, objects: make(map[string]types.Object)}
		return pp, pp.parse()
	}
	return nil, fmt.Errorf("package named '%s' could not be loaded from the go build system. Please make sure the current folder contains the go.mod for the target package, or that its go.mod is in a parent directory", packageName)
}

func (pp *packageParser) parse() error {
	fileSet := token.NewFileSet()
	importer := importer.Default()

	// If no field names are requested, use all
	if pp.fieldNames == nil {
		pp.fieldNames = pp.pkg.Scope().Names()
	}

	for _, typeName := range pp.fieldNames {
		typ, obj, err := lookupType(pp.pkg.Scope(), typeName)
		if err != nil {
			return err
		}
		var mtyp *TypeDef
		if _, ok := typ.Underlying().(*types.Struct); ok {
			mtyp = newStructDef(fileSet, importer, typ, pp.packagePath)
		} else {
			mtyp = newPrimitiveDef(fileSet, importer, typ, pp.packagePath)
		}
		pp.results = append(pp.results, mtyp)
		pp.objects[typeName] = obj
	}
	return nil
}

func (pp *packageParser) TypeDefs() []*TypeDef {
	return pp.results
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

func (pp *packageParser) TypeDefSourceCode() ([]byte, error) {
	in := NewImportNamer(pp.pkg)
	structs := make([]string, 0)
	for _, def := range pp.objects {
		defstring := types.ObjectString(def, in.Name)
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
