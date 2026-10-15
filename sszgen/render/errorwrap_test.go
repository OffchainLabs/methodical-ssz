package render

import (
	"strings"
	"testing"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// TestErrorWrapFieldName locks in that every nested SSZ method call
// (UnmarshalSSZ, MarshalSSZTo, HashTreeRootWith) wraps its error with the name
// of the field where it occurred, so a failure deep in a nested type reports a
// field path rather than a bare sentinel. Both a directly-nested container
// field (Bar) and a collection element (Bars, whose element failure wraps with
// the owning field) are covered.
func TestErrorWrapFieldName(t *testing.T) {
	inner := &gentypes.ValueContainer{
		Name:    "Inner",
		Package: "github.com/example/foo",
		Contents: []gentypes.ContainerField{
			{Key: "X", Value: &gentypes.ValueUint{Name: "uint64", Size: 64}},
		},
	}
	outer := &gentypes.ValueContainer{
		Name:    "Outer",
		Package: "github.com/example/foo",
		Contents: []gentypes.ContainerField{
			{Key: "Bar", Value: inner},
			{Key: "Bars", Value: &gentypes.ValueList{MaxSize: 10, ElementValue: &gentypes.ValuePointer{Referent: inner}}},
		},
	}

	got, err := Render("github.com/example/foo", "", []gentypes.ValRep{outer})
	if err != nil {
		t.Fatalf("render.Render: %v", err)
	}
	src := string(got)

	// A directly-nested container field wraps each of the three method calls.
	for _, want := range []string{
		`return fmt.Errorf("Bar: %w", err)`, // MarshalSSZTo / HashTreeRootWith
		`return nil, fmt.Errorf("Bar: %w", err)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated code missing wrap %q", want)
		}
	}
	// A collection element failure wraps with the owning field name, not the
	// loop temp variable.
	if !strings.Contains(src, `fmt.Errorf("Bars: %w", err)`) {
		t.Errorf("generated code missing element wrap for field Bars")
	}
}
