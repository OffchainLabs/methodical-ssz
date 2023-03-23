package sszgen

import (
	"fmt"
	"go/types"

	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/pkg/errors"
)

type FieldParserOpt func(*FieldParser)

func WithDisableDelegation() FieldParserOpt {
	return func(p *FieldParser) {
		p.disableDelegation = true
	}
}

func ParseTypeDef(typ *TypeDef, opts ...FieldParserOpt) (gentypes.ValRep, error) {
	p := &FieldParser{}
	for _, o := range opts {
		o(p)
	}
	if typ.IsStruct {
		vr := &gentypes.ValueContainer{
			Name:    typ.Name,
			Package: typ.orig.Obj().Pkg().Path(),
		}
		for _, f := range typ.Fields {
			rep, err := p.expand(f)
			if err != nil {
				return nil, err
			}
			vr.Append(f.name, rep)
		}
		return vr, nil
	}
	// PrimitiveType is stored in Fields[0]
	rep, err := p.expand(typ.Fields[0])
	if err != nil {
		return nil, err
	}
	vr := &gentypes.ValueOverlay{
		Name:       typ.Name,
		Package:    rep.PackagePath(),
		Underlying: rep,
	}
	return vr, nil
}

type FieldParser struct {
	disableDelegation bool
}

func (p *FieldParser) expand(f *FieldDef) (gentypes.ValRep, error) {
	switch ty := f.typ.(type) {
	case *types.Array:
		size := int(ty.Len())
		return p.expandArray([]*SSZDimension{{VectorLength: &size}}, f)
	case *types.Slice:
		return p.expandArrayHead(f)
	case *types.Pointer:
		vr, err := p.expand(&FieldDef{name: f.name, tag: f.tag, typ: ty.Elem(), pkg: f.pkg})
		if err != nil {
			return nil, err
		}
		v := &gentypes.ValuePointer{Referent: vr}
		if !p.disableDelegation {
			v.Interfaces = interfaces.NewSSZSupportMap(ty)
		}
		return v, nil
	case *types.Struct:
		container := gentypes.ValueContainer{
			Name:    f.name,
			Package: f.pkg.Path(),
		}
		if !p.disableDelegation {
			container.Interfaces = interfaces.NewSSZSupportMap(ty)
		}
		for i := 0; i < ty.NumFields(); i++ {
			field := ty.Field(i)
			if field.Name() == "" || !field.Exported() {
				continue
			}
			rep, err := p.expand(&FieldDef{name: field.Name(), tag: ty.Tag(i), typ: field.Type(), pkg: field.Pkg()})
			if err != nil {
				return nil, err
			}
			container.Append(f.name, rep)
		}
		return &container, nil
	case *types.Named:
		exp, err := p.expand(&FieldDef{name: ty.Obj().Name(), tag: f.tag, typ: ty.Underlying(), pkg: f.pkg})
		switch ty.Underlying().(type) {
		case *types.Struct:
			return exp, err
		default:
			v := &gentypes.ValueOverlay{
				Name:       ty.Obj().Name(),
				Package:    ty.Obj().Pkg().Path(),
				Underlying: exp,
			}
			if !p.disableDelegation {
				v.Interfaces = interfaces.NewSSZSupportMap(ty)
			}
			return v, err
		}
	case *types.Basic:
		return p.expandIdent(ty.Kind(), ty.Name())
	default:
		return nil, fmt.Errorf("unsupported type for %v with name: %v", ty, f.name)
	}
}

func (p *FieldParser) expandArrayHead(f *FieldDef) (gentypes.ValRep, error) {
	dims, err := extractSSZDimensions(fmt.Sprintf("`%v`", f.tag))
	if err != nil {
		return nil, errors.Wrapf(err, "name=%s, package=%s, tag=%s", f.name, f.pkg.Path(), f.tag)
	}
	return p.expandArray(dims, f)
}

func (p *FieldParser) expandArray(dims []*SSZDimension, f *FieldDef) (gentypes.ValRep, error) {
	if len(dims) == 0 {
		return nil, fmt.Errorf("do not have dimension information for type %v", f.name)
	}
	d := dims[0]
	var (
		elv  gentypes.ValRep
		err  error
		elem types.Type
	)
	isArray := false
	// at this point f.typ is either and array or a slice
	if arr, ok := f.typ.(*types.Array); ok {
		isArray = true
		elem = arr.Elem()
	} else if arr, ok := f.typ.(*types.Slice); ok {
		elem = arr.Elem()
	} else {
		return nil, fmt.Errorf("invalid typ in expand array: %v with name: %v ", f.typ, f.name)
	}

	// Only expand the inner array if it is not a named type
	if _, ok := elem.(*types.Named); !ok && len(dims) > 1 {
		elv, err = p.expandArray(dims[1:], &FieldDef{name: f.name, typ: elem.Underlying(), pkg: f.pkg})
		if err != nil {
			return nil, err
		}
	} else {
		elv, err = p.expand(&FieldDef{name: f.name, tag: f.tag, typ: elem, pkg: f.pkg})
		if err != nil {
			return nil, err
		}
	}

	if d.IsVector() {
		return &gentypes.ValueVector{
			IsArray:      isArray,
			ElementValue: elv,
			Size:         d.VectorLen(),
		}, nil
	}
	if d.IsList() {
		return &gentypes.ValueList{
			ElementValue: elv,
			MaxSize:      d.ListLen(),
		}, nil
	}
	return nil, nil
}

func (p *FieldParser) expandIdent(ident types.BasicKind, name string) (gentypes.ValRep, error) {
	switch ident {
	case types.Bool:
		return &gentypes.ValueBool{Name: name}, nil
	case types.Byte:
		return &gentypes.ValueByte{Name: name}, nil
	case types.Uint16:
		return &gentypes.ValueUint{Size: 16, Name: name}, nil
	case types.Uint32:
		return &gentypes.ValueUint{Size: 32, Name: name}, nil
	case types.Uint64:
		return &gentypes.ValueUint{Size: 64, Name: name}, nil
		/*
			case "uint128":
				return &gentypes.ValueUint{Size: 128, Name: ident.name}, nil
			case "uint256":
				return &gentypes.ValueUint{Size: 256, Name: ident.name}, nil
		*/
	default:
		return nil, fmt.Errorf("unknown ident: %v", name)
	}
}
