package backend

import (
	"os"
	"strings"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

func TestGenerateUnmarshalSSZ(t *testing.T) {
	t.Skip("fixtures need to be updated")
	b, err := os.ReadFile("testdata/TestGenerateUnmarshalSSZ.expected")
	if err != nil {
		t.Fatal(err)
	}
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	if !ok {
		t.Fatal("testFixBeaconState failed to assert to type *types.ValueContainer")
	}
	gc := &generateContainer{ValueContainer: vc, targetPackage: ""}
	code, err := GenerateUnmarshalSSZ(gc)
	if err != nil {
		t.Fatalf("err from GenerateUnmarshalSSZ=%v", err)
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

func TestUnmarshalSteps(t *testing.T) {
	fixturePath := "testdata/TestUnmarshalSteps.expected"
	b, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := normalizeFixtureBytes(b)
	if err != nil {
		t.Fatal(err)
	}

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	if !ok {
		t.Fatal("testFixBeaconState failed to assert to type *types.ValueContainer")
	}
	gc := &generateContainer{ValueContainer: vc, targetPackage: ""}
	ums := gc.unmarshalSteps()
	if len(ums) != 21 {
		t.Fatalf("expected 21 unmarshal steps, got %d", len(ums))
	}
	if ums[15].nextVariable.fieldNumber != ums[16].fieldNumber {
		t.Fatalf("expected field numbers to match, got %d and %d", ums[15].nextVariable.fieldNumber, ums[16].fieldNumber)
	}

	gotRaw := strings.Join([]string{ums.fixedSlices(), "", ums.variableSlices(gc.fixedOffset())}, "\n")
	actual, err := normalizeFixtureString(gotRaw)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}
