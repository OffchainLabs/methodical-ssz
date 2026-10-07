package sszgen

import (
	"fmt"
	"go/token"
	"go/types"
	"os"

	"github.com/OffchainLabs/methodical-ssz/sszgen/config"
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
	object      types.Object
	cfg         config.TypeConfig
}

func (t *TypeDef) NaturalLeaves() int {
	return len(t.Fields)
}

// FieldDef represents a field of the intermediate marshaling type.
type FieldDef struct {
	name string
	typ  types.Type
	tag  string
	pkg  *types.Package
}

func newStructDef(fs *token.FileSet, ps PathScoper, typ *types.Named, obj types.Object) *TypeDef {
	name := typ.Obj().Name()
	mtyp := &TypeDef{
		Name:        name,
		PackageName: ps.Path(),
		IsStruct:    true,
		fs:          fs,
		orig:        typ,
		cfg:         ps.TypeConfig(name),
		object:      obj,
	}

	styp := typ.Underlying().(*types.Struct)
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

func newPrimitiveDef(fs *token.FileSet, ps PathScoper, typ *types.Named, obj types.Object) *TypeDef {
	name := typ.Obj().Name()
	mtyp := &TypeDef{
		Name:        name,
		PackageName: ps.Path(),
		IsStruct:    false,
		fs:          fs,
		orig:        typ,
		cfg:         ps.TypeConfig(name),
		object:      obj,
	}

	fd := &FieldDef{
		name: typ.Underlying().String(),
		typ:  typ.Underlying(),
		tag:  "",
		pkg:  typ.Obj().Pkg(),
	}
	mtyp.Fields = append(mtyp.Fields, fd)
	return mtyp
}
