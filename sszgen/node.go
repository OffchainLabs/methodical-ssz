package sszgen

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
	"reflect"
	"strings"

	sszgenTypes "github.com/kasey/methodical-ssz/sszgen/types"
)

// TypeDef represents the intermediate struct type used during marshaling.
// This is the input data to all the Go code templates.
type TypeDef struct {
	Name        string
	PackageName string
	Fields      []*FieldDef
	fs          *token.FileSet
	orig        *types.Named
	override    *types.Named
	scope       *fileScope
}

// FieldDef represents a field of the intermediate marshaling type.
type FieldDef struct {
	name     string
	typ      types.Type
	origTyp  types.Type
	tag      string
	function *types.Func // map to a function instead of a field
	ValRep   sszgenTypes.ValRep
}

func newMarshalerType(fs *token.FileSet, imp types.Importer, typ *types.Named, packageName string) *TypeDef {
	mtyp := &TypeDef{Name: typ.Obj().Name(), fs: fs, orig: typ, PackageName: packageName}
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

		mf := &FieldDef{
			name:    f.Name(),
			typ:     f.Type(),
			origTyp: f.Type(),
			tag:     styp.Tag(i),
		}

		mtyp.Fields = append(mtyp.Fields, mf)
	}

	return mtyp
}

// findFunction returns a function with `name` that accepts no arguments
// and returns a single value that is convertible to the given to type.
func findFunction(typ *types.Named, name string, to types.Type) (*types.Func, types.Type) {
	for i := 0; i < typ.NumMethods(); i++ {
		fun := typ.Method(i)
		if fun.Name() != name || !fun.Exported() {
			continue
		}
		sign := fun.Type().(*types.Signature)
		if sign.Params().Len() != 0 || sign.Results().Len() != 1 {
			continue
		}
		if err := checkConvertible(sign.Results().At(0).Type(), to); err == nil {
			return fun, sign.Results().At(0).Type()
		}
	}
	return nil, nil
}

// loadOverrides sets field types of the intermediate marshaling type from
// matching fields of otyp.
func (mtyp *TypeDef) loadOverrides(otyp *types.Named) error {
	s := otyp.Underlying().(*types.Struct)
	for i := 0; i < s.NumFields(); i++ {
		of := s.Field(i)
		if of.Anonymous() || !of.Exported() {
			return fmt.Errorf("%v: field override type cannot have embedded or unexported fields", mtyp.fs.Position(of.Pos()))
		}
		f := mtyp.fieldByName(of.Name())
		if f == nil {
			// field not defined in original type, check if it maps to a suitable function and add it as an override
			if fun, retType := findFunction(mtyp.orig, of.Name(), of.Type()); fun != nil {
				f = &FieldDef{name: fun.Name(), origTyp: retType, typ: of.Type(), function: fun, tag: s.Tag(i)}
				mtyp.Fields = append(mtyp.Fields, f)
			} else {
				return fmt.Errorf("%v: no matching field or function for %s in original type %s", mtyp.fs.Position(of.Pos()), of.Name(), mtyp.Name)
			}
		}
		if err := checkConvertible(of.Type(), f.origTyp); err != nil {
			return fmt.Errorf("%v: invalid field override: %v", mtyp.fs.Position(of.Pos()), err)
		}
		f.typ = of.Type()
	}
	mtyp.scope.addReferences(s)
	mtyp.override = otyp
	return nil
}

func (mtyp *TypeDef) fieldByName(name string) *FieldDef {
	for _, f := range mtyp.Fields {
		if f.name == name {
			return f
		}
	}
	return nil
}

// isRequired returns whether the field is required when decoding the given format.
func (mf *FieldDef) isRequired(format string) bool {
	rtag := reflect.StructTag(mf.tag)
	req := rtag.Get("gencodec") == "required"
	// Fields with json:"-" must be treated as optional. This also works
	// for the other supported formats.
	return req && !strings.HasPrefix(rtag.Get(format), "-")
}

// encodedName returns the alternative field name assigned by the format's struct tag.
func (mf *FieldDef) encodedName(format string) string {
	val := reflect.StructTag(mf.tag).Get(format)
	if comma := strings.Index(val, ","); comma != -1 {
		val = val[:comma]
	}
	if val == "" || val == "-" {
		return uncapitalize(mf.name)
	}
	return val
}

func uncapitalize(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}
