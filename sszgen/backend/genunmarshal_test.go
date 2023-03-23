package backend

import (
	"os"
	"strings"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGenerateUnmarshalSSZ(t *testing.T) {
	t.Skip("fixtures need to be updated")
	b, err := os.ReadFile("testdata/TestGenerateUnmarshalSSZ.expected")
	require.NoError(t, err)
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := &generateContainer{ValueContainer: vc, targetPackage: ""}
	code, err := GenerateUnmarshalSSZ(gc)
	require.NoError(t, err)
	require.Equal(t, 4, len(code.imports))
	actual, err := normalizeFixtureString(code.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestUnmarshalSteps(t *testing.T) {
	fixturePath := "testdata/TestUnmarshalSteps.expected"
	b, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	expected, err := normalizeFixtureBytes(b)
	require.NoError(t, err)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := &generateContainer{ValueContainer: vc, targetPackage: ""}
	ums := gc.unmarshalSteps()
	require.Equal(t, 21, len(ums))
	require.Equal(t, ums[15].nextVariable.fieldNumber, ums[16].fieldNumber)

	gotRaw := strings.Join([]string{ums.fixedSlices(), "", ums.variableSlices(gc.fixedOffset())}, "\n")
	actual, err := normalizeFixtureString(gotRaw)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
