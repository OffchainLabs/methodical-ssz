package testutil

import (
	"testing"

	"github.com/kasey/methodical-ssz/sszgen/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRenderIntermediate(t *testing.T) {
	s := &types.ValueContainer{
		Name:    "testing",
		Package: "github.com/prysmaticlabs/derp",
		Contents: []types.ContainerField{
			{
				Key: "OverlayUint",
				Value: &types.ValuePointer{Referent: &types.ValueOverlay{
					Name:    "FakeContainer",
					Package: "github.com/prysmaticlabs/derp/derp",
					Underlying: &types.ValueUint{
						Name: "uint8",
						Size: 8,
					},
				},
				},
			},
		},
	}
	expected := ""
	actual, err := RenderIntermediate(s)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
