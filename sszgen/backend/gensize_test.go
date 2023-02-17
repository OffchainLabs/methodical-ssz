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
	gc, err := GenerateSizeSSZ(&generateContainer{ty, ""})
	require.NoError(t, err)
	require.Equal(t, 4, len(gc.imports))
	actual, err := normalizeFixtureString(gc.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
