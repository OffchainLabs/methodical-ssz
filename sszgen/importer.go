package sszgen

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
)

// NewImporter returns an importer that loads packages with the given build
// tags. When more than one source variant of a package is tag-gated in a
// single directory (e.g. mainnet `//go:build !minimal` vs minimal
// `//go:build minimal` files), the tags select which variant the type checker
// sees. Passing no tags loads the default (untagged) variant.
func NewImporter(buildTags ...string) *GoPkgImporter {
	return &GoPkgImporter{
		cache:     make(map[string]*packages.Package),
		buildTags: buildTags,
	}
}

type GoPkgImporter struct {
	cache     map[string]*packages.Package
	buildTags []string
}

func (imp *GoPkgImporter) Load(name string) (*packages.Package, error) {
	if p, ok := imp.cache[name]; ok {
		return p, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports,
	}
	if len(imp.buildTags) > 0 {
		cfg.BuildFlags = []string{"-tags=" + strings.Join(imp.buildTags, ",")}
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
