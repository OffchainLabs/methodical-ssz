package sszgen

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	"github.com/pkg/errors"
)

type VirtualFile struct {
	name     string
	contents string
}

type VirtualPathScoper struct {
	vfiles []VirtualFile
	files  []*ast.File
	scope  *types.Scope
	path   string
}

func NewVirtualPathScoper(pkgName string, vfs ...VirtualFile) (*VirtualPathScoper, error) {
	vps := &VirtualPathScoper{
		vfiles: vfs,
		files:  make([]*ast.File, len(vfs)),
		path:   pkgName,
	}
	fset := token.NewFileSet()
	for i, vf := range vfs {
		f, err := parser.ParseFile(fset, vf.name, vf.contents, 0)
		if err != nil {
			return nil, err
		}
		vps.files[i] = f
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(pkgName, fset, vps.files, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error from conf.Check")
	}
	vps.scope = pkg.Scope()
	return vps, nil
}

func (vps *VirtualPathScoper) Path() string {
	return vps.path
}

func (vps *VirtualPathScoper) Scope() *types.Scope {
	return vps.scope
}
