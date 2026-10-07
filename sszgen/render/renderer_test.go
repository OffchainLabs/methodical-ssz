package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// update regenerates the checked-in golden when set: `go test ./sszgen/render
// -run TestRenderGolden -update`. Updating it is an explicit, reviewable act,
// not an automatic pass.
var update = flag.Bool("update", false, "update golden files")

// exampleContainer exercises the breadth the ops need: a fixed scalar, a fixed
// byte-vector, a variable list of fixed elements, a list of nested lists
// (variable elements), and a list of pointers to a variable container (the
// beacon-attestations shape: list-of-variable + pointer + container recursion).
func exampleContainer() *gentypes.ValueContainer {
	inner := &gentypes.ValueContainer{
		Name:    "Inner",
		Package: "github.com/example/foo",
		Contents: []gentypes.ContainerField{
			{Key: "X", Value: &gentypes.ValueList{MaxSize: 64, ElementValue: &gentypes.ValueByte{Name: "byte"}}},
		},
	}
	// fixedInner is fixed-size so a vector of it stays fixed-size (a different
	// codegen path from vectors of variable containers).
	fixedInner := &gentypes.ValueContainer{
		Name:    "FixedInner",
		Package: "github.com/example/foo",
		Contents: []gentypes.ContainerField{
			{Key: "Y", Value: &gentypes.ValueUint{Name: "uint64", Size: 64}},
		},
	}
	slot := &gentypes.ValueOverlay{Name: "Slot", Package: "github.com/example/foo", Underlying: &gentypes.ValueUint{Name: "uint64", Size: 64}}
	return &gentypes.ValueContainer{
		Name:    "Example",
		Package: "github.com/example/foo",
		Contents: []gentypes.ContainerField{
			{Key: "A", Value: &gentypes.ValueUint{Name: "uint64", Size: 64}},
			{Key: "B", Value: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}}},
			{Key: "C", Value: &gentypes.ValueList{MaxSize: 1024, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}},
			{Key: "D", Value: &gentypes.ValueList{MaxSize: 256, ElementValue: &gentypes.ValueList{MaxSize: 64, ElementValue: &gentypes.ValueByte{Name: "byte"}}}},
			{Key: "E", Value: &gentypes.ValueList{MaxSize: 128, ElementValue: &gentypes.ValuePointer{Referent: inner}}},
			{Key: "F", Value: &gentypes.ValuePointer{Referent: inner}}, // direct variable pointer: nil-init guard
			// htr-specific shapes:
			{Key: "G", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}}}},                                                           // vector of 32-byte vectors: element Append
			{Key: "H", Value: &gentypes.ValueOverlay{Name: "MyInt", Package: "github.com/example/foo", Underlying: &gentypes.ValueUint{Name: "uint64", Size: 64}}},                                                      // overlay-of-uint: coerce
			{Key: "I", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}},                                                                                              // packed uint vector
			{Key: "J", Value: &gentypes.ValueOverlay{Name: "Bitvector4", Package: "github.com/OffchainLabs/go-bitfield", Underlying: &gentypes.ValueVector{Size: 1, ElementValue: &gentypes.ValueByte{Name: "byte"}}}},  // bitvector: overlay->vector coerce
			{Key: "K", Value: &gentypes.ValueList{MaxSize: 256, ElementValue: &gentypes.ValueByte{Name: "byte"}}},                                                                                                       // byte list: AppendBytes32 special case
			{Key: "L", Value: &gentypes.ValueOverlay{Name: "Bitlist", Package: "github.com/OffchainLabs/go-bitfield", Underlying: &gentypes.ValueList{MaxSize: 2048, ElementValue: &gentypes.ValueByte{Name: "byte"}}}}, // bitlist: PutBitlist
			// HTR shapes with element casts and per-element delegation:
			{Key: "M", Value: &gentypes.ValueVector{Size: 4, ElementValue: slot}},                                         // vector of overlay-of-uint: cast inside element loop
			{Key: "N", Value: &gentypes.ValueList{MaxSize: 1024, ElementValue: slot}},                                     // list of overlay-of-uint: cast inside element loop
			{Key: "O", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValuePointer{Referent: fixedInner}}}, // vector of fixed containers: per-element delegation, not a direct call
			// vectors of variable-sized elements:
			{Key: "P", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueList{MaxSize: 64, ElementValue: &gentypes.ValueByte{Name: "byte"}}}}, // vector of byte lists
			{Key: "Q", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValuePointer{Referent: inner}}},                                           // vector of variable containers
			// shapes with working codegen paths that had no fixture coverage:
			{Key: "R", Value: &gentypes.ValueBool{Name: "bool"}},
			{Key: "S", Value: &gentypes.ValueUint{Name: "uint16", Size: 16}},
			{Key: "T", Value: &gentypes.ValueUint{Name: "uint32", Size: 32}},
			{Key: "U", Value: &gentypes.ValueList{MaxSize: 32, ElementValue: &gentypes.ValueList{MaxSize: 16, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}}},                                                                                         // list of uint64 lists: depth-unique temps
			{Key: "V", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueList{MaxSize: 16, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}}},                                                                                           // vector of uint64 lists: depth-unique temps
			{Key: "W", Value: &gentypes.ValueOverlay{Name: "Root", Package: "github.com/example/foo", Underlying: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}}}},                                                                 // non-bitfield byte-vector overlay
			{Key: "X", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueOverlay{Name: "Bitvector8", Package: "github.com/OffchainLabs/go-bitfield", Underlying: &gentypes.ValueVector{Size: 1, ElementValue: &gentypes.ValueByte{Name: "byte"}}}}}, // vector of bitvectors
			// test-matrix.md Phase 2: shapes unbroken by the Phase 2 fixes:
			{Key: "Y", Value: &gentypes.ValueVector{Size: 8, ElementValue: &gentypes.ValueUint{Name: "uint32", Size: 32}}},                                                                                                                                              // packed uint32 vector (AppendUint32)
			{Key: "Z", Value: &gentypes.ValueList{MaxSize: 100, ElementValue: &gentypes.ValueUint{Name: "uint16", Size: 16}}},                                                                                                                                           // packed uint16 list (AppendUint16 + pad)
			{Key: "AA", Value: &gentypes.ValueVector{Size: 2, ElementValue: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}}},                                                                                               // vector of uint64 vectors
			{Key: "AB", Value: &gentypes.ValueList{MaxSize: 32, ElementValue: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueUint{Name: "uint64", Size: 64}}}},                                                                                             // list of uint64 vectors
			{Key: "AC", Value: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}, IsArray: true}},                                                                                                                                         // [32]byte array vector
			{Key: "AD", Value: &gentypes.ValueByte{Name: "byte"}},                                                                                                                                                                                                       // plain byte field
			{Key: "AE", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueBool{Name: "bool"}}},                                                                                                                                                         // packed bool vector
			{Key: "AF", Value: &gentypes.ValueList{MaxSize: 100, ElementValue: &gentypes.ValueBool{Name: "bool"}}},                                                                                                                                                      // packed bool list
			{Key: "AG", Value: &gentypes.ValueList{MaxSize: 4, ElementValue: &gentypes.ValueOverlay{Name: "Bitlist", Package: "github.com/OffchainLabs/go-bitfield", Underlying: &gentypes.ValueList{MaxSize: 2048, ElementValue: &gentypes.ValueByte{Name: "byte"}}}}}, // list of bitlists: per-element root + outer mixin
			{Key: "AH", Value: &gentypes.ValueVector{Size: 2, ElementValue: &gentypes.ValueOverlay{Name: "Bitlist", Package: "github.com/OffchainLabs/go-bitfield", Underlying: &gentypes.ValueList{MaxSize: 2048, ElementValue: &gentypes.ValueByte{Name: "byte"}}}}},  // vector of bitlists
			// uint256 via github.com/holiman/uint256.Int (recognized by the
			// frontend like the go-bitfield types; [4]uint64 little-endian limbs):
			{Key: "AI", Value: &gentypes.ValueUint{Name: "Int", Package: "github.com/holiman/uint256", Size: 256}},                                                // uint256 scalar
			{Key: "AJ", Value: &gentypes.ValueList{MaxSize: 8, ElementValue: &gentypes.ValueUint{Name: "Int", Package: "github.com/holiman/uint256", Size: 256}}}, // list of uint256
			{Key: "AK", Value: &gentypes.ValueVector{Size: 2, ElementValue: &gentypes.ValueUint{Name: "Int", Package: "github.com/holiman/uint256", Size: 256}}},  // vector of uint256
			// byte arrays in element position (TypeName must render [N]byte):
			{Key: "AL", Value: &gentypes.ValueList{MaxSize: 16, ElementValue: &gentypes.ValueVector{Size: 48, ElementValue: &gentypes.ValueByte{Name: "byte"}, IsArray: true}}}, // list of [48]byte
			{Key: "AM", Value: &gentypes.ValueVector{Size: 4, ElementValue: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}, IsArray: true}}},   // vector of [32]byte
			// named overlay of a byte array (`type ArrayRoot [32]byte`): no
			// coerce conversion — a named array slices directly:
			{Key: "AN", Value: &gentypes.ValueOverlay{Name: "ArrayRoot", Package: "github.com/example/foo", Underlying: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}, IsArray: true}}},
			{Key: "AO", Value: &gentypes.ValueList{MaxSize: 8, ElementValue: &gentypes.ValueOverlay{Name: "ArrayRoot", Package: "github.com/example/foo", Underlying: &gentypes.ValueVector{Size: 32, ElementValue: &gentypes.ValueByte{Name: "byte"}, IsArray: true}}}},
		},
	}
}

// Union types are deliberately unsupported (the frontend never produces them);
// every op must refuse loudly rather than silently emit nothing.
func TestUnionPanics(t *testing.T) {
	union := &gentypes.ValueUnion{}
	cases := []struct {
		name string
		gen  func()
	}{
		{"size", func() { core.Dispatch(sizeOp{acc: sizeVar}, union, "c.X", nil) }},
		{"marshal", func() { core.Dispatch(marshalOp{}, union, "c.X", nil) }},
		{"unmarshal", func() { core.Dispatch(unmarshalOp{}, union, unmarshalRef{Cast: identity}, nil) }},
		{"htr", func() { core.Dispatch(htrOp{}, union, "c.X", nil) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected a panic for a union type")
				}
			}()
			c.gen()
		})
	}
}

// Top-level overlays generate no methods: Render skips them rather than
// erroring, producing a file with only the package header.
func TestRenderTopLevelOverlaySkipped(t *testing.T) {
	ov := &gentypes.ValueOverlay{Name: "MyInt", Package: "github.com/example/foo", Underlying: &gentypes.ValueUint{Name: "uint64", Size: 64}}
	got, err := Render("github.com/example/foo", "", []gentypes.ValRep{ov})
	if err != nil {
		t.Fatalf("render.Render: %v", err)
	}
	if strings.Contains(string(got), "func ") {
		t.Fatalf("expected no generated methods for a top-level overlay, got:\n%s", got)
	}
}

// Render requires a package path (or an override) before doing any work.
func TestRenderEmptyPackagePath(t *testing.T) {
	if _, err := Render("", "", nil); err == nil {
		t.Fatal("expected an error for an empty packagePath")
	}
}

// TestDefaultMethodSets pins the canonical method-set order, which is part of the
// generated file's shape.
func TestDefaultMethodSets(t *testing.T) {
	got := DefaultMethodSets()
	want := []string{"size", "marshal", "unmarshal", "htr", "htr-progressive"}
	if len(got) != len(want) {
		t.Fatalf("want %d method sets, got %d", len(want), len(got))
	}
	for i, n := range want {
		if got[i].Name != n {
			t.Fatalf("method set %d: want %q, got %q", i, n, got[i].Name)
		}
	}
}

// TestRenderGolden is the regression gate for the full Render path: the output
// for exampleContainer (which exercises every SSZ shape across all four method
// sets) must match the checked-in golden.
func TestRenderGolden(t *testing.T) {
	got, err := Render("github.com/example/foo", "", []gentypes.ValRep{exampleContainer()})
	if err != nil {
		t.Fatalf("render.Render: %v", err)
	}

	golden := filepath.Join("testdata", "example.ssz.go.expected")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
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
}
