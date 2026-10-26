package testutil

import (
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
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
	expected := `package values

var testing types.ValRep = &types.ValueContainer{Name: "testing", Package: "github.com/prysmaticlabs/derp", Contents: []types.ContainerField{{Key: "OverlayUint", Value: &types.ValuePointer{Referent: &types.ValueOverlay{Name: "FakeContainer", Package: "github.com/prysmaticlabs/derp/derp", Underlying: &types.ValueUint{Name: "uint8", Size: 8}}}}}}
`
	actual, err := RenderIntermediate(s)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}
