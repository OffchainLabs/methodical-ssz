package backend

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

func TestGenerateSizeSSZ(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateSizeSSZ.expected")
	if err != nil {
		t.Fatal(err)
	}
	expected := string(b)

	ty, ok := testFixBeaconState.(*types.ValueContainer)
	if !ok {
		t.Fatal("testFixBeaconState failed to assert to type *types.ValueContainer")
	}
	inm := NewImportNamer("", nil)
	gc, err := GenerateSizeSSZ(&generateContainer{ValueContainer: ty, targetPackage: "", importNamer: inm})
	if err != nil {
		t.Fatalf("err from GenerateSizeSSZ=%v", err)
	}
	// the size code for BeaconState is all fixed values and calls to values inside loops, so it can safely assume nothing needs
	// to be initialized.
	// TODO: Add a test case for size code for a type like BeaconBlockBodyBellatrix that needs to init for safety
	// (ie actually requires imports)
	if len(gc.imports) != 0 {
		t.Fatalf("expected 0 imports, got %d", len(gc.imports))
	}
	actual, err := normalizeFixtureString(gc.blocks[0])
	if err != nil {
		t.Fatalf("err from normalizeFixtureString=%v", err)
	}
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}
