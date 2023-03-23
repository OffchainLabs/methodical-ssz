package backend

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

// cases left to satisfy:
// list-vector-byte
func TestGenerateHashTreeRoot(t *testing.T) {
	t.Skip("fixtures need to be updated")
	b, err := os.ReadFile("testdata/TestGenerateHashTreeRoot.expected")
	require.NoError(t, err)
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := &generateContainer{ValueContainer: vc, targetPackage: ""}
	code, err := GenerateHashTreeRoot(gc)
	require.NoError(t, err)
	require.Equal(t, 4, len(code.imports))
	actual, err := normalizeFixtureString(code.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
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
	require.Equal(t, expected, actual)
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
	require.Equal(t, expected, actual)
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
	require.Equal(t, expected, actual)
}
