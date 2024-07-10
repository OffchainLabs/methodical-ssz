package sszgen

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func NewImporter() *Importer {
	return &Importer{
		cache: make(map[string]*packages.Package),
	}
}

type Importer struct {
	cache map[string]*packages.Package
}

func (imp *Importer) Load(name string) (*packages.Package, error) {
	if p, ok := imp.cache[name]; ok {
		return p, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports,
	}
	pkgs, err := packages.Load(cfg, name)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if pkg.ID != name {
			continue
		}

		imp.cache[name] = pkg
		return imp.cache[name], nil
	}
	return nil, fmt.Errorf("package named '%s' could not be loaded from the go build system. Please make sure the current folder contains the go.mod for the target package, or that its go.mod is in a parent directory", name)
}

func (imp *Importer) Import(name string) (*types.Package, error) {
	p, err := imp.Load(name)
	if err != nil {
		return nil, err
	}
	return p.Types, nil
}
