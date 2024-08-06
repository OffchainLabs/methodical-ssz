package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// progressiveContainer exercises the progressive merkleization shapes: a
// marked container (with an inactive position), progressive lists of basic,
// byte, and composite elements, a progressive bitlist, and a nested marked
// container (whose standard HashTreeRootWith — called by the parent's field
// putter — redirects to its progressive form).
func progressiveContainer() *gentypes.ValueContainer {
	nested := &gentypes.ValueContainer{
		Name:         "NestedProg",
		Package:      "github.com/example/foo",
		ActiveFields: []bool{true, true},
		Contents: []gentypes.ContainerField{
			{Key: "A", Value: &gentypes.ValueUint{Name: "uint64", Size: 64}},
			{Key: "B", Value: &gentypes.ValueList{Progressive: true, ElementValue: &gentypes.ValueByte{Name: "byte"}}}, // ProgressiveByteList
		},
	}
	return &gentypes.ValueContainer{
		Name:    "Prog",
		Package: "github.com/example/foo",
		// five fields with one inactive position (index 2): [1,1,0,1,1,1]
		ActiveFields: []bool{true, true, false, true, true, true},
		Contents: []gentypes.ContainerField{
			{Key: "A", Value: &gentypes.ValueUint{Name: "uint64", Size: 64}},
			{Key: "B", Value: &gentypes.ValueList{Progressive: true, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}},                                                                                          // ProgressiveList[uint64]
			{Key: "C", Value: &gentypes.ValueList{Progressive: true, ElementValue: &gentypes.ValuePointer{Referent: nested}}},                                                                                               // ProgressiveList of containers
			{Key: "D", Value: &gentypes.ValueOverlay{Name: "Bitlist", Package: "github.com/OffchainLabs/go-bitfield", Underlying: &gentypes.ValueList{Progressive: true, ElementValue: &gentypes.ValueByte{Name: "byte"}}}}, // ProgressiveBitlist
			{Key: "E", Value: &gentypes.ValuePointer{Referent: nested}},                                                                                                                                                     // nested progressive container
		},
	}
}

// TestRenderProgressive gates the progressive method set: marked containers
// get ProgressiveHashTreeRoot[With] plus standard-name wrappers; progressive
// collections merkleize with no limit. Serialization output is identical to
// the regular forms (the spec serializes progressive types like their
// non-progressive counterparts, minus the max checks).
func TestRenderProgressive(t *testing.T) {
	got, err := Render("github.com/example/foo", "", []gentypes.ValRep{progressiveContainer()})
	if err != nil {
		t.Fatalf("render.Render: %v", err)
	}

	golden := filepath.Join("testdata", "progressive.ssz.go.expected")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("reading golden (run with -update to create it): %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("Render output differs from golden %s (run with -update if the change is intended):\n--- want ---\n%s\n--- got ---\n%s", golden, want, got)
	}

	// The wrapper contract: the standard names must delegate to the
	// progressive forms for marked types.
	for _, frag := range []string{
		"func (c *Prog) HashTreeRoot() ([32]byte, error) {\n\treturn c.ProgressiveHashTreeRoot()\n}",
		"func (c *Prog) HashTreeRootWith(hh *ssz.Hasher) error {\n\treturn c.ProgressiveHashTreeRootWith(hh)\n}",
		"var activeFieldsProg = []byte{0b00111011}",
		"hh.MerkleizeProgressiveWithActiveFields(indx, activeFieldsProg)",
	} {
		if !strings.Contains(string(got), frag) {
			t.Fatalf("missing expected fragment:\n%s\nin output:\n%s", frag, got)
		}
	}
}
