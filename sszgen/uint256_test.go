package sszgen

import (
	"go/token"
	"go/types"
	"testing"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// The frontend recognizes github.com/holiman/uint256.Int (by package+name, like
// the go-bitfield types) as the SSZ uint256 basic type. Value fields and
// collection elements become ValueUint{Size: 256}; pointer fields still wrap it
// in a ValuePointer, where delegation to the package's own SSZ methods can
// attach. The type is constructed synthetically because the virtual scoper's
// importer cannot resolve module dependencies.
func TestUint256Recognition(t *testing.T) {
	pkg := types.NewPackage("github.com/holiman/uint256", "uint256")
	obj := types.NewTypeName(token.NoPos, pkg, "Int", nil)
	named := types.NewNamed(obj, types.NewArray(types.Typ[types.Uint64], 4), nil)

	assertUint256 := func(name string, vr gentypes.ValRep) {
		t.Helper()
		vu, ok := vr.(*gentypes.ValueUint)
		if !ok {
			t.Fatalf("%s: want *ValueUint, got %T", name, vr)
		}
		if vu.Size != gentypes.Uint256 {
			t.Fatalf("%s: want size 256, got %d", name, vu.Size)
		}
		if vu.Package != "github.com/holiman/uint256" || vu.Name != "Int" {
			t.Fatalf("%s: unexpected type identity %s.%s", name, vu.Package, vu.Name)
		}
	}

	p := &FieldParser{}

	vr, err := p.expand(&FieldDef{name: "Balance", typ: named, pkg: pkg})
	if err != nil {
		t.Fatal(err)
	}
	assertUint256("value field", vr)

	vr, err = p.expand(&FieldDef{name: "Balances", typ: types.NewSlice(named), tag: `ssz-max:"8"`, pkg: pkg})
	if err != nil {
		t.Fatal(err)
	}
	list, ok := vr.(*gentypes.ValueList)
	if !ok {
		t.Fatalf("slice field: want *ValueList, got %T", vr)
	}
	assertUint256("list element", list.ElementValue)

	vr, err = p.expand(&FieldDef{name: "Pointer", typ: types.NewPointer(named), pkg: pkg})
	if err != nil {
		t.Fatal(err)
	}
	ptr, ok := vr.(*gentypes.ValuePointer)
	if !ok {
		t.Fatalf("pointer field: want *ValuePointer, got %T", vr)
	}
	assertUint256("pointer referent", ptr.Referent)
}
