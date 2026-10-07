package core

import (
	"go/types"
	"testing"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// countOp is a trivial Op[int] used to verify the generic visitor machinery
// compiles and that Dispatch routes per kind and recurses. It counts the nodes in
// a ValRep tree.
type countOp struct{}

func (countOp) Name() string                                                { return "count" }
func (countOp) Delegates(*GenContext) *types.Interface                      { return nil }
func (countOp) Delegate(gentypes.ValRep, string, *GenContext) int           { return -1 }
func (countOp) Bool(*gentypes.ValueBool, string, *GenContext) int           { return 1 }
func (countOp) Byte(*gentypes.ValueByte, string, *GenContext) int           { return 1 }
func (countOp) Uint(*gentypes.ValueUint, string, *GenContext) int           { return 1 }
func (countOp) Container(*gentypes.ValueContainer, string, *GenContext) int { return 1 }
func (countOp) Union(*gentypes.ValueUnion, string, *GenContext) int         { return 1 }
func (countOp) Vector(v *gentypes.ValueVector, ref string, ctx *GenContext) int {
	return 1 + Dispatch(countOp{}, v.ElementValue, ref, ctx)
}
func (countOp) List(v *gentypes.ValueList, ref string, ctx *GenContext) int {
	return 1 + Dispatch(countOp{}, v.ElementValue, ref, ctx)
}
func (countOp) Overlay(v *gentypes.ValueOverlay, ref string, ctx *GenContext) int {
	return 1 + Dispatch(countOp{}, v.Underlying, ref, ctx)
}
func (countOp) Pointer(v *gentypes.ValuePointer, ref string, ctx *GenContext) int {
	return 1 + Dispatch(countOp{}, v.Referent, ref, ctx)
}

func TestDispatchRoutesAndRecurses(t *testing.T) {
	// list -> vector -> uint  => 3 nodes
	vr := &gentypes.ValueList{
		ElementValue: &gentypes.ValueVector{
			Size:         4,
			ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64},
		},
	}
	if got := Dispatch(countOp{}, vr, "x", nil); got != 3 {
		t.Fatalf("want 3 nodes, got %d", got)
	}
}
