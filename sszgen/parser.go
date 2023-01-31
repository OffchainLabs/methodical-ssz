package sszgen

import (
	"fmt"
	"go/importer"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type packageParser struct {
	// configuration
	packagePath string
	fieldNames  []string
	// parsed values
	pkg     *types.Package
	results []*TypeDef
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

		pp := &packageParser{packagePath: pkg.ID, fieldNames: fieldNames, pkg: pkg.Types}
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
		typ, err := lookupType(pp.pkg.Scope(), typeName)
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
	}
	return nil
}

func (pp *packageParser) TypeDefs() []*TypeDef {
	return pp.results
}
