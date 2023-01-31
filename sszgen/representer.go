package sszgen

import (
	"fmt"
	"go/types"

	sszgenTypes "github.com/kasey/methodical-ssz/sszgen/types"
	"github.com/pkg/errors"
)

func ParseStruct(typ *TypeDef) (sszgenTypes.ValRep, error) {
	vr := &sszgenTypes.ValueContainer{
		Name:    typ.Name,
		Package: typ.PackageName,
	}
	for _, f := range typ.Fields {
		// this filters out internal protobuf fields, but also serializers like us
		// can safely ignore unexported fields in general. We also ignore embedded
		// fields because I'm not sure if we should support them yet.
		if f.name == "" {
			continue
		}
		rep, err := expand(f, typ.PackageName)
		if err != nil {
			return nil, err
		}
		vr.Append(f.name, rep)
	}
	return vr, nil

}

func expand(f *FieldDef, pkg string) (sszgenTypes.ValRep, error) {
	switch ty := f.typ.(type) {
	case *types.Array:
		return expandArrayHead(f, pkg)
	case *types.Slice:
		return expandArrayHead(f, pkg)
	case *types.Pointer:
		vr, err := expand(&FieldDef{name: f.name, tag: f.tag, typ: ty.Elem()}, pkg)
		if err != nil {
			return nil, err
		}
		return &sszgenTypes.ValuePointer{Referent: vr}, nil
	case *types.Basic:
		return expandIdent(ty.Kind(), f.name)
	case *types.Named:
		return expand(&FieldDef{name: f.name, tag: f.tag, typ: ty.Underlying()}, pkg)
	case *types.Struct:
		container := sszgenTypes.ValueContainer{
			Name:    f.name,
			Package: pkg,
		}
		for i := 0; i < ty.NumFields(); i++ {
			field := ty.Field(i)
			if field.Name() == "" || !field.Exported() {
				continue
			}
			rep, err := expand(&FieldDef{name: field.Name(), tag: ty.Tag(i), typ: field.Type()}, pkg)
			if err != nil {
				return nil, err
			}
			container.Append(f.name, rep)
		}
		return &container, nil
	default:
		return nil, fmt.Errorf("unsupported type for %v with name: %v", ty, f.name)
	}
}

func expandArrayHead(f *FieldDef, pkg string) (sszgenTypes.ValRep, error) {
	dims, err := extractSSZDimensions(fmt.Sprintf("`%v`", f.tag))
	if err != nil {
		return nil, errors.Wrapf(err, "name=%s, package=%s, tag=%s", f.name, pkg, f.tag)
	}
	return expandArray(dims, f, pkg)
}

func expandArray(dims []*SSZDimension, f *FieldDef, pkg string) (sszgenTypes.ValRep, error) {
	if len(dims) == 0 {
		return nil, fmt.Errorf("do not have dimension information for type %v", f.name)
	}
	d := dims[0]
	var (
		elv  sszgenTypes.ValRep
		err  error
		elem types.Type
	)
	// at this point f.typ is either and array or a slice
	if arr, ok := f.typ.(*types.Array); ok {
		elem = arr.Elem()
	} else if arr, ok := f.typ.(*types.Slice); ok {
		elem = arr.Elem()
	} else {
		return nil, fmt.Errorf("invalid typ in expand array: %v with name: %v ", f.typ, f.name)
	}

	switch elt := elem.(type) {
	case *types.Array:
		elv, err = expandArray(dims[1:], &FieldDef{typ: elt.Elem()}, pkg)
		if err != nil {
			return nil, err
		}
	default:
		elv, err = expand(&FieldDef{name: f.name, tag: f.tag, typ: elt}, pkg)
		if err != nil {
			return nil, err
		}
	}

	if d.IsVector() {
		return &sszgenTypes.ValueVector{
			ElementValue: elv,
			Size:         d.VectorLen(),
		}, nil
	}
	if d.IsList() {
		return &sszgenTypes.ValueList{
			ElementValue: elv,
			MaxSize:      d.ListLen(),
		}, nil
	}
	return nil, nil
}

func expandIdent(ident types.BasicKind, name string) (sszgenTypes.ValRep, error) {
	switch ident {
	case types.Bool:
		return &sszgenTypes.ValueBool{Name: name}, nil
	case types.Byte:
		return &sszgenTypes.ValueByte{Name: name}, nil
	case types.Uint16:
		return &sszgenTypes.ValueUint{Size: 16, Name: name}, nil
	case types.Uint32:
		return &sszgenTypes.ValueUint{Size: 32, Name: name}, nil
	case types.Uint64:
		return &sszgenTypes.ValueUint{Size: 64, Name: name}, nil
		/*
			case "uint128":
				return &sszgenTypes.ValueUint{Size: 128, Name: ident.name}, nil
			case "uint256":
				return &sszgenTypes.ValueUint{Size: 256, Name: ident.name}, nil
		*/
	default:
		return nil, fmt.Errorf("unknown ident: %v", name)
	}
}
