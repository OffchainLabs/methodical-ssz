package sszgen

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
)

// TypeDef represents the intermediate struct type used during marshaling.
// This is the input data to all the Go code templates.
type TypeDef struct {
	Name        string
	PackageName string
	IsStruct    bool
	Fields      []*FieldDef
	fs          *token.FileSet
	orig        *types.Named
	scope       *fileScope
	object      types.Object
}

// FieldDef represents a field of the intermediate marshaling type.
type FieldDef struct {
	name string
	typ  types.Type
	tag  string
	pkg  *types.Package
	// declared as a slice pointer for unambiguous zero-value
	dims *[]*SSZDimension
}

func newStructDef(fs *token.FileSet, imp types.Importer, typ *types.Named, packageName string) *TypeDef {
	mtyp := &TypeDef{
		Name:        typ.Obj().Name(),
		PackageName: packageName,
		IsStruct:    true,
		fs:          fs,
		orig:        typ,
	}

	styp := typ.Underlying().(*types.Struct)
	mtyp.scope = newFileScope(imp, typ.Obj().Pkg())
	mtyp.scope.addReferences(styp)

	// Add packages which are always needed.
	mtyp.scope.addImport("encoding/json")
	mtyp.scope.addImport("errors")

	for i := 0; i < styp.NumFields(); i++ {
		f := styp.Field(i)
		if !f.Exported() {
			continue
		}
		if f.Anonymous() {
			fmt.Fprintf(os.Stderr, "Warning: ignoring embedded field %s\n", f.Name())
			continue
		}

		switch ftyp := f.Type().(type) {
		case *types.Pointer:
			fn, ok := ftyp.Elem().(*types.Named)
			if ok {
				mf := &FieldDef{
					name: f.Name(),
					typ:  f.Type(),
					tag:  styp.Tag(i),
					pkg:  fn.Obj().Pkg(),
				}
				mtyp.Fields = append(mtyp.Fields, mf)
				continue
			}
		}
		mf := &FieldDef{
			name: f.Name(),
			typ:  f.Type(),
			tag:  styp.Tag(i),
			pkg:  f.Pkg(),
		}
		mtyp.Fields = append(mtyp.Fields, mf)
	}
	return mtyp
}

func newPrimitiveDef(fs *token.FileSet, imp types.Importer, typ *types.Named, packageName string) *TypeDef {
	mtyp := &TypeDef{
		Name:        typ.Obj().Name(),
		PackageName: packageName,
		IsStruct:    false,
		fs:          fs,
		orig:        typ,
	}
	mtyp.scope = newFileScope(imp, typ.Obj().Pkg())
	mtyp.scope.addReferences(typ.Underlying())

	// Add packages which are always needed.
	mtyp.scope.addImport("encoding/json")
	mtyp.scope.addImport("errors")

	fd := &FieldDef{
		name: typ.Underlying().String(),
		typ:  typ.Underlying(),
		tag:  "",
		pkg:  typ.Obj().Pkg(),
	}
	mtyp.Fields = append(mtyp.Fields, fd)
	return mtyp
}
