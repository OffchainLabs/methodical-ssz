package backend

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// cases left to satisfy:
// list-vector-byte
func TestGenerateHashTreeRoot(t *testing.T) {
	t.Skip("fixtures need to be updated")
	b, err := os.ReadFile("testdata/TestGenerateHashTreeRoot.expected")
	if err != nil {
		t.Fatal(err)
	}
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	if !ok {
		t.Fatal("testFixBeaconState failed to assert to type *types.ValueContainer")
	}
	if ok != true {
		t.Fatal("failed to cast testFixBeaconState to ValueContainer")
	}
	gc := &generateContainer{ValueContainer: vc, targetPackage: ""}
	code, err := GenerateHashTreeRoot(gc)
	if err != nil {
		t.Fatalf("err from GenerateHashTreeRoot=%v", err)
	}
	if len(code.imports) != 4 {
		t.Fatalf("expected 4 imports, got %d", len(code.imports))
	}
	actual, err := normalizeFixtureString(code.blocks[0])
	if err != nil {
		t.Fatalf("err from normalizeFixtureString=%v", err)
	}
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestHTROverlayCoerce(t *testing.T) {
	pkg := "derp"
	expected := "hh.PutUint64(uint64(b.Slot))"
	val := &types.ValueOverlay{
		Name:    "",
		Package: pkg,
		Underlying: &types.ValueUint{
			Name:    "uint64",
			Size:    64,
			Package: pkg,
		},
	}
	gv := &generateOverlay{ValueOverlay: val, targetPackage: pkg}
	actual := gv.generateHTRPutter("b.Slot")
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestHTRContainer(t *testing.T) {
	t.Skip("fixtures need to be updated")
	pkg := "derp"
	expected := `if err := b.Fork.HashTreeRootWith(hh); err != nil {
		return err
	}`
	val := &types.ValueContainer{}
	gv := &generateContainer{ValueContainer: val, targetPackage: pkg}
	actual := gv.generateHTRPutter("b.Fork")
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestHTRByteVector(t *testing.T) {
	t.Skip("fixtures need to be updated")
	pkg := "derp"
	fieldName := "c.GenesisValidatorsRoot"
	expected := `{
	if len(c.GenesisValidatorsRoot) != 32 {
		return ssz.ErrVectorLength
	}
	hh.PutBytes(c.GenesisValidatorsRoot)
}`
	val := &types.ValueVector{
		ElementValue: &types.ValueByte{},
		Size:         32,
	}
	gv := &generateVector{
		valRep:        val,
		targetPackage: pkg,
	}
	actual := gv.generateHTRPutter(fieldName)
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}
