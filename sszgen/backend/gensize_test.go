package backend

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGenerateSizeSSZ(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateSizeSSZ.expected")
	require.NoError(t, err)
	expected := string(b)

	ty, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	inm := NewImportNamer("", nil)
	gc, err := GenerateSizeSSZ(&generateContainer{ValueContainer: ty, targetPackage: "", importNamer: inm})
	require.NoError(t, err)
	// the size code for BeaconState is all fixed values and calls to values inside loops, so it can safely assume nothing needs
	// to be initialized.
	// TODO: Add a test case for size code for a type like BeaconBlockBodyBellatrix that needs to init for safety
	// (ie actually requires imports)
	require.Equal(t, 0, len(gc.imports))
	actual, err := normalizeFixtureString(gc.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
