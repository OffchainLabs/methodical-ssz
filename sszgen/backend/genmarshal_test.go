package backend

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGenerateMarshalSSZ(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateMarshalSSZ.expected")
	require.NoError(t, err)
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	inm := NewImportNamer("", nil)
	gc := &generateContainer{ValueContainer: vc, targetPackage: "", importNamer: inm}
	code, err := GenerateMarshalSSZ(gc)
	require.NoError(t, err)
	require.Equal(t, 2, len(inm.aliases))
	actual, err := normalizeFixtureString(code.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
