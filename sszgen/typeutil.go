// Copyright 2017 Felix Lange <fjl@twurst.com>.
// Use of this source code is governed by the MIT license,
// which can be found in the LICENSE file.

package sszgen

import (
	"fmt"
	"go/types"
	"io"
	"sort"
	"strconv"

	"github.com/pkg/errors"
)

// walkNamedTypes runs the callback for all named types contained in the given type.
func walkNamedTypes(typ types.Type, callback func(*types.Named)) {
	switch typ := typ.(type) {
	case *types.Basic:
	case *types.Chan:
		walkNamedTypes(typ.Elem(), callback)
	case *types.Map:
		walkNamedTypes(typ.Key(), callback)
		walkNamedTypes(typ.Elem(), callback)
	case *types.Named:
		callback(typ)
	case *types.Pointer:
		walkNamedTypes(typ.Elem(), callback)
	case *types.Slice:
	case *types.Array:
		walkNamedTypes(typ.Elem(), callback)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			walkNamedTypes(typ.Field(i).Type(), callback)
		}
	case *types.Interface:
		if typ.NumMethods() > 0 {
			panic("BUG: can't walk non-empty interface")
		}
	default:
		panic(fmt.Errorf("BUG: can't walk %T", typ))
	}
}

func lookupType(scope *types.Scope, name string) (*types.Named, types.Object, error) {
	if name == "" {
		return nil, nil, errors.Wrap(errors.New("no such identifier"), "empty name lookup")
	}
	obj := scope.Lookup(name)
	if obj == nil {
		return nil, nil, errors.Wrap(errors.New("no such identifier"), name)
	}
	typ, ok := obj.(*types.TypeName)
	if !ok {
		return nil, nil, errors.Wrap(errors.New("not a type"), name)
	}
	return typ.Type().(*types.Named), obj, nil
}

// fileScope tracks imports and other names at file scope.
type fileScope struct {
	imports       []*types.Package
	importsByName map[string]*types.Package
	importNames   map[string]string
	otherNames    map[string]bool // non-package identifiers
	pkg           *types.Package
	imp           types.Importer
}

func newFileScope(imp types.Importer, pkg *types.Package) *fileScope {
	return &fileScope{otherNames: make(map[string]bool), pkg: pkg, imp: imp}
}

func (s *fileScope) writeImportDecl(w io.Writer) {
	fmt.Fprintln(w, "import (")
	for _, pkg := range s.imports {
		if s.importNames[pkg.Path()] != pkg.Name() {
			fmt.Fprintf(w, "\t%s %q\n", s.importNames[pkg.Path()], pkg.Path())
		} else {
			fmt.Fprintf(w, "\t%q\n", pkg.Path())
		}
	}
	fmt.Fprintln(w, ")")
}

// addImport loads a package and adds it to the import set.
func (s *fileScope) addImport(path string) {
	pkg, err := s.imp.Import(path)
	if err != nil {
		panic(fmt.Errorf("can't import %q: %v", path, err))
	}
	s.insertImport(pkg)
	s.rebuildImports()
}

// addReferences marks all names referenced by typ as used.
func (s *fileScope) addReferences(typ types.Type) {
	walkNamedTypes(typ, func(nt *types.Named) {
		pkg := nt.Obj().Pkg()
		if pkg == s.pkg {
			s.otherNames[nt.Obj().Name()] = true
		} else if pkg != nil {
			s.insertImport(nt.Obj().Pkg())
		}
	})
	s.rebuildImports()
}

// insertImport adds pkg to the list of known imports.
// This method should not be used directly because it doesn't
// rebuild the import name cache.
func (s *fileScope) insertImport(pkg *types.Package) {
	i := sort.Search(len(s.imports), func(i int) bool {
		return s.imports[i].Path() >= pkg.Path()
	})
	if i < len(s.imports) && s.imports[i] == pkg {
		return
	}
	s.imports = append(s.imports[:i], append([]*types.Package{pkg}, s.imports[i:]...)...)
}

// rebuildImports caches the names of imported packages.
func (s *fileScope) rebuildImports() {
	s.importNames = make(map[string]string)
	s.importsByName = make(map[string]*types.Package)
	for _, pkg := range s.imports {
		s.maybeRenameImport(pkg)
	}
}

func (s *fileScope) maybeRenameImport(pkg *types.Package) {
	name := pkg.Name()
	for i := 0; s.isNameTaken(name); i++ {
		name = pkg.Name()
		if i > 0 {
			name += strconv.Itoa(i - 1)
		}
	}
	s.importNames[pkg.Path()] = name
	s.importsByName[name] = pkg
}

// isNameTaken reports whether the given name is used by an import or other identifier.
func (s *fileScope) isNameTaken(name string) bool {
	return s.importsByName[name] != nil || s.otherNames[name] || types.Universe.Lookup(name) != nil
}

// qualify is a types.Qualifier that prepends the (possibly renamed) package name of
// imported types to a type name.
func (s *fileScope) qualify(pkg *types.Package) string {
	if pkg == s.pkg {
		return ""
	}
	return s.packageName(pkg.Path())
}

func (s *fileScope) packageName(path string) string {
	name, ok := s.importNames[path]
	if !ok {
		panic("BUG: missing package " + path)
	}
	return name
}
