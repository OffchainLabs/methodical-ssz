package sszgen

import (
	"fmt"
	"go/types"

	sszgenTypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/pkg/errors"
)

func ParseTypeDef(typ *TypeDef) (sszgenTypes.ValRep, error) {
	if typ.IsStruct {
		vr := &sszgenTypes.ValueContainer{
			Name:    typ.Name,
			Package: typ.orig.Obj().Pkg().Path(),
		}
		for _, f := range typ.Fields {
			rep, err := expand(f)
			if err != nil {
				return nil, err
			}
			vr.Append(f.name, rep)
		}
		return vr, nil
	}
	// PrimitiveType is stored in Fields[0]
	rep, err := expand(typ.Fields[0])
	if err != nil {
		return nil, err
	}
	vr := &sszgenTypes.ValueOverlay{
		Name:       typ.Name,
		Package:    rep.PackagePath(),
		Underlying: rep,
	}
	return vr, nil
}

func expand(f *FieldDef) (sszgenTypes.ValRep, error) {
	switch ty := f.typ.(type) {
	case *types.Array:
		size := int(ty.Len())
		return expandArray([]*SSZDimension{{VectorLength: &size}}, f)
	case *types.Slice:
		return expandArrayHead(f)
	case *types.Pointer:
		// If the struct pointer implements the fastssz interfaces, use those for
		// serialization and merkleization instead of generating code.
		if ty, ok := ty.Elem().(*types.Named); ok {
			if types.Implements(f.typ, fastsszMarshaler) && types.Implements(f.typ, fastsszUnmarshaler) && types.Implements(f.typ, fastsszLightHasher) {
				return &sszgenTypes.ValueContainer{
					Name:      ty.Obj().Name(),
					Package:   ty.Obj().Pkg().Path(),
					LightHash: !types.Implements(f.typ, fastsszFullHasher),
				}, nil
			}
		}
		vr, err := expand(&FieldDef{name: f.name, tag: f.tag, typ: ty.Elem(), pkg: f.pkg})
		if err != nil {
			return nil, err
		}
		return &sszgenTypes.ValuePointer{Referent: vr}, nil
	case *types.Struct:
		container := sszgenTypes.ValueContainer{
			Name:    f.name,
			Package: f.pkg.Path(),
		}
		for i := 0; i < ty.NumFields(); i++ {
			field := ty.Field(i)
			if field.Name() == "" || !field.Exported() {
				continue
			}
			rep, err := expand(&FieldDef{name: field.Name(), tag: ty.Tag(i), typ: field.Type(), pkg: field.Pkg()})
			if err != nil {
				return nil, err
			}
			container.Append(f.name, rep)
		}
		return &container, nil
	case *types.Named:
		// If the struct value implements the fastssz interfaces, use those for
		// serialization and merkleization instead of generating code. Some of
		// the methods might be on the value and some on the pointer, find both.
		var (
			marshaler   = types.Implements(ty, fastsszMarshaler) || types.Implements(types.NewPointer(ty), fastsszMarshaler)
			unmarshaler = types.Implements(ty, fastsszUnmarshaler) || types.Implements(types.NewPointer(ty), fastsszUnmarshaler)
			lightHasher = types.Implements(ty, fastsszLightHasher) || types.Implements(types.NewPointer(ty), fastsszLightHasher)
			fullHasher  = types.Implements(ty, fastsszFullHasher) || types.Implements(types.NewPointer(ty), fastsszFullHasher)
		)
		if marshaler && unmarshaler && lightHasher {
			return &sszgenTypes.ValueContainer{
				Name:      ty.Obj().Name(),
				Package:   ty.Obj().Pkg().Path(),
				Value:     true,
				LightHash: !fullHasher,
			}, nil
		}
		exp, err := expand(&FieldDef{name: ty.Obj().Name(), tag: f.tag, typ: ty.Underlying(), pkg: f.pkg})
		switch ty.Underlying().(type) {
		case *types.Struct:
			return exp, err
		default:
			return &sszgenTypes.ValueOverlay{
				Name:       ty.Obj().Name(),
				Package:    ty.Obj().Pkg().Path(),
				Underlying: exp,
			}, err
		}

	case *types.Basic:
		return expandIdent(ty.Kind(), ty.Name())
	default:
		return nil, fmt.Errorf("unsupported type for %v with name: %v", ty, f.name)
	}
}

func expandArrayHead(f *FieldDef) (sszgenTypes.ValRep, error) {
	dims, err := extractSSZDimensions(fmt.Sprintf("`%v`", f.tag))
	if err != nil {
		return nil, errors.Wrapf(err, "name=%s, package=%s, tag=%s", f.name, f.pkg.Path(), f.tag)
	}
	return expandArray(dims, f)
}

func expandArray(dims []*SSZDimension, f *FieldDef) (sszgenTypes.ValRep, error) {
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

	if len(dims) > 1 {
		elv, err = expandArray(dims[1:], &FieldDef{typ: elem.Underlying(), pkg: f.pkg})
		if err != nil {
			return nil, err
		}
	} else {
		elv, err = expand(&FieldDef{name: f.name, tag: f.tag, typ: elem, pkg: f.pkg})
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
