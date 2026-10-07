package render

import (
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// A vector of variable-sized elements contributes a 4-byte offset plus the
// element's encoded size, per element — the same loop as a variable list.
func TestSizeVariableVector(t *testing.T) {
	vr := &gentypes.ValueVector{
		Size:         4,
		ElementValue: &gentypes.ValueList{MaxSize: 8, ElementValue: &gentypes.ValueByte{Name: "byte"}},
	}
	got := core.Dispatch(sizeOp{acc: sizeVar}, vr, "c.X", nil)
	want := `for _, o := range c.X {
		size += 4
		size += len(o)
	}`
	if got.Variable != want {
		t.Fatalf("unexpected size fragment:\n--- want ---\n%s\n--- got ---\n%s", want, got.Variable)
	}
	if got.Init != "" {
		t.Fatalf("expected no init expression, got %q", got.Init)
	}
}
