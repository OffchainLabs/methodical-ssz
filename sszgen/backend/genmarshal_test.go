package backend

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

func TestGenerateMarshalSSZ(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateMarshalSSZ.expected")
	if err != nil {
		t.Fatal(err)
	}
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	if !ok {
		t.Fatal("testFixBeaconState failed to assert to type *types.ValueContainer")
	}
	inm := NewImportNamer("", nil)
	gc := &generateContainer{ValueContainer: vc, targetPackage: "", importNamer: inm}
	code, err := GenerateMarshalSSZ(gc)
	if err != nil {
		t.Fatalf("err from GenerateMarshalSSZ=%v", err)
	}
	if len(inm.aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(inm.aliases))
	}
	actual, err := normalizeFixtureString(code.blocks[0])
	if err != nil {
		t.Fatalf("err from normalizeFixtureString=%v", err)
	}
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}
