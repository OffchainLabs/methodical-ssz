package sszgen

import (
	"fmt"
	"go/types"

	"github.com/OffchainLabs/methodical-ssz/sszgen/config"
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
			rep, err := p.expandField(f, typ.cfg.Fields[f.name])
			if err != nil {
				return nil, err
			}
			vr.Append(f.name, rep)
		}
		if typ.cfg.Progressive != nil {
			af, err := typ.cfg.Progressive.ActiveFields(len(typ.Fields))
			if err != nil {
				return nil, fmt.Errorf("type %s: %w", typ.Name, err)
			}
			vr.ActiveFields = af
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
	typeConfig        *config.GeneratorConfig
	// ifaces is the delegation interface set used to build support maps. Its
	// pointers are identity keys shared with the render GenContext; the
	// default is interfaces.DefaultSet(), injectable via WithInterfaceSet for
	// a plugin targeting its own interfaces.
	ifaces *interfaces.Set
}

// WithInterfaceSet injects the delegation interface set used to build the
// ValReps' support maps. The same Set instance must be supplied to the render
// GenContext — the interface pointers are identity keys.
func WithInterfaceSet(s *interfaces.Set) FieldParserOpt {
	return func(p *FieldParser) {
		p.ifaces = s
	}
}

// supportMap builds the delegation support map for ty from the parser's
// interface set, defaulting to interfaces.DefaultSet on first use.
func (p *FieldParser) supportMap(ty types.Type) (map[*types.Interface]bool, error) {
	if p.ifaces == nil {
		s, err := interfaces.DefaultSet()
		if err != nil {
			return nil, err
		}
		p.ifaces = s
	}
	return p.ifaces.SupportMap(ty), nil
}

// expandField expands a struct field, applying any yaml field-config override
// (progressive collections are declared in the generator config rather than
// struct tags — they have no spec limit, but an ssz-max tag on the field is
// retained as a limit on how large of a list unmarshal will allow.
func (p *FieldParser) expandField(f *FieldDef, fc config.FieldConfig) (gentypes.ValRep, error) {
	switch fc.Type {
	case "":
		return p.expand(f)
	case config.FieldTypeProgressiveList, config.FieldTypeProgressiveByteList:
		return p.expandProgressiveList(f, fc)
	case config.FieldTypeProgressiveBitlist:
		return p.expandProgressiveBitlist(f)
	default:
		return nil, fmt.Errorf("field %s: unknown field config type %q", f.name, fc.Type)
	}
}

// expandProgressiveList expands a slice-typed field as an SSZ ProgressiveList.
func (p *FieldParser) expandProgressiveList(f *FieldDef, fc config.FieldConfig) (gentypes.ValRep, error) {
	if _, ok := f.typ.(*types.Slice); !ok {
		return nil, fmt.Errorf("field %s: %s requires a slice type, got %v", f.name, fc.Type, f.typ)
	}
	// Expand as a bounded list first to resolve element SSZ dimensions against
	// the tag, then mark it progressive. MaxSize is preserved: progressive
	// lists have no spec limit, but the ssz-max limit is enforced when
	// unmarshaling to prevent malicious inputs from triggering unbounded
	// allocation.
	vr, err := p.expand(f)
	if err != nil {
		return nil, err
	}
	list, ok := vr.(*gentypes.ValueList)
	if !ok {
		return nil, fmt.Errorf("field %s: %s requires a list type, got %v", f.name, fc.Type, f.typ)
	}
	if fc.Type == config.FieldTypeProgressiveByteList {
		if _, isByte := list.ElementValue.(*gentypes.ValueByte); !isByte {
			return nil, fmt.Errorf("field %s: ProgressiveByteList requires a byte slice, got %v", f.name, f.typ)
		}
	}
	// Apply an optional element override for nested progressive collections
	// whose element has no named Go type to delegate to.
	elem, err := applyElementConfig(f.name, list.ElementValue, fc.Element)
	if err != nil {
		return nil, err
	}
	return &gentypes.ValueList{ElementValue: elem, Progressive: true, MaxSize: list.MaxSize}, nil
}

// applyElementConfig rewrites an already-expanded element value to its
// progressive form (recursing for deeper nesting), honoring a nested FieldConfig.
func applyElementConfig(field string, elem gentypes.ValRep, ec *config.FieldConfig) (gentypes.ValRep, error) {
	if ec == nil {
		return elem, nil
	}
	switch ec.Type {
	case config.FieldTypeProgressiveList, config.FieldTypeProgressiveByteList:
		list, ok := elem.(*gentypes.ValueList)
		if !ok {
			return nil, fmt.Errorf("field %s: element %s requires a list element, got %T", field, ec.Type, elem)
		}
		if ec.Type == config.FieldTypeProgressiveByteList {
			if _, isByte := list.ElementValue.(*gentypes.ValueByte); !isByte {
				return nil, fmt.Errorf("field %s: element ProgressiveByteList requires a byte slice, got %v", field, elem)
			}
		}
		inner, err := applyElementConfig(field, list.ElementValue, ec.Element)
		if err != nil {
			return nil, err
		}
		return &gentypes.ValueList{ElementValue: inner, Progressive: true, MaxSize: list.MaxSize}, nil
	default:
		return nil, fmt.Errorf("field %s: unsupported element config type %q", field, ec.Type)
	}
}

// expandProgressiveBitlist expands a go-bitfield Bitlist-style field (a named
// type whose underlying type is a byte slice) as an SSZ ProgressiveBitlist.
func (p *FieldParser) expandProgressiveBitlist(f *FieldDef) (gentypes.ValRep, error) {
	named, ok := f.typ.(*types.Named)
	if !ok {
		return nil, fmt.Errorf("field %s: ProgressiveBitlist requires a named bitlist type, got %v", f.name, f.typ)
	}
	slice, ok := named.Underlying().(*types.Slice)
	if !ok {
		return nil, fmt.Errorf("field %s: ProgressiveBitlist requires a byte-slice-backed type, got %v", f.name, f.typ)
	}
	basic, ok := slice.Elem().(*types.Basic)
	if !ok || basic.Kind() != types.Byte {
		return nil, fmt.Errorf("field %s: ProgressiveBitlist requires a byte-slice-backed type, got %v", f.name, f.typ)
	}
	// A progressive bitlist has no spec limit and needs no tag, but when the
	// field carries an ssz-max tag (in bits) it is retained so unmarshaling
	// enforces the same limit as the non-progressive form.
	maxSize := 0
	if dims, err := extractSSZDimensions(fmt.Sprintf("`%v`", f.tag)); err == nil && len(dims) > 0 && dims[0].IsList() {
		maxSize = dims[0].ListLen()
	}
	v := &gentypes.ValueOverlay{
		Name:       named.Obj().Name(),
		Package:    named.Obj().Pkg().Path(),
		Underlying: &gentypes.ValueList{ElementValue: &gentypes.ValueByte{Name: "byte"}, Progressive: true, MaxSize: maxSize},
	}
	if !p.disableDelegation {
		ifs, err := p.supportMap(named)
		if err != nil {
			return nil, err
		}
		v.Interfaces = ifs
	}
	return v, nil
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
			ifs, err := p.supportMap(ty)
			if err != nil {
				return nil, err
			}
			v.Interfaces = ifs
			// the pointer's SatisfiesInterface needs to recognize the
			// unmarshaler key to refuse referent fall-through for it
			v.Unmarshaler = p.ifaces.Unmarshaler
		}
		return v, nil
	case *types.Struct:
		container := gentypes.ValueContainer{
			Name:    f.name,
			Package: f.pkg.Path(),
		}
		if !p.disableDelegation {
			ifs, err := p.supportMap(ty)
			if err != nil {
				return nil, err
			}
			container.Interfaces = ifs
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
		pkg := ty.Obj().Pkg()
		// Recognize github.com/holiman/uint256.Int (by package+name) as the SSZ
		// uint256 basic type. Pointer fields still delegate via the pointer
		// wrapper; this native form serves value fields and collection elements.
		if pkg != nil && pkg.Path() == "github.com/holiman/uint256" && ty.Obj().Name() == "Int" {
			return &gentypes.ValueUint{Name: "Int", Package: pkg.Path(), Size: gentypes.Uint256}, nil
		}
		exp, err := p.expand(&FieldDef{name: ty.Obj().Name(), tag: f.tag, typ: ty.Underlying(), pkg: pkg})
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
				ifs, err := p.supportMap(ty)
				if err != nil {
					return nil, err
				}
				v.Interfaces = ifs
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
		// TODO: uint128 unimplemented (no blessed Go representation).
	default:
		return nil, fmt.Errorf("unknown ident: %v", name)
	}
}
