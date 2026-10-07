package ssz

import (
	"encoding/hex"
	"strings"
	"testing"
)

// twoField is a minimal HashRoot type (Container{a,b: uint64}) for exercising
// the TraceValue entry point.
type twoField struct{ a, b uint64 }

func (v twoField) HashTreeRootWith(h *Hasher) error {
	indx := h.Index()
	h.PutUint64(v.a)
	h.PutUint64(v.b)
	h.Merkleize(indx)
	return nil
}
func (v twoField) HashTreeRoot() ([32]byte, error) { return HashWithDefaultHasher(v) }

func TestTraceValueAndFlatEmit(t *testing.T) {
	res, err := TraceValue(twoField{1, 2}, "AB")
	if err != nil {
		t.Fatalf("TraceValue: %v", err)
	}
	const wantRoot = "0xff55c97976a840b4ced964ed49e3794594ba3f675238b5fd25d282b60f70a194"
	if hexRoot(res.Root) != wantRoot {
		t.Fatalf("root %s want %s", hexRoot(res.Root), wantRoot)
	}
	var flat strings.Builder
	if err := res.WriteFlat(&flat); err != nil {
		t.Fatalf("WriteFlat: %v", err)
	}
	got := flat.String()
	for _, want := range []string{
		"trace_version: 1",
		`type: "AB"`,
		`root: "` + wantRoot + `"`,
		"nodes:",
		`  1: "` + wantRoot + `"`,
		`  2: "0x0100000000000000000000000000000000000000000000000000000000000000"`,
		`  3: "0x0200000000000000000000000000000000000000000000000000000000000000"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("flat output missing %q; got:\n%s", want, got)
		}
	}
}

func TestTraceValueYAMLEmit(t *testing.T) {
	res, err := TraceValue(twoField{1, 2}, "AB")
	if err != nil {
		t.Fatalf("TraceValue: %v", err)
	}
	var y strings.Builder
	if err := res.WriteYAML(&y); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}
	got := y.String()
	for _, want := range []string{
		"elide_zero_subtrees: true",
		"tree:",
		"  g: 1",
		"  left:",
		"    g: 2",
		`    leaf: "0x0100000000000000000000000000000000000000000000000000000000000000"`,
		"  right:",
		"    g: 3",
		`    leaf: "0x0200000000000000000000000000000000000000000000000000000000000000"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("yaml output missing %q; got:\n%s", want, got)
		}
	}
}

// Cross-library validation: reproduce the exact remerkleable oracle roots.
// Matching these proves the Go reconstruction agrees with the oracle on packing,
// limits, depth, mix-ins, and zero elision — end to end — without needing the
// Python harness at runtime.

// List[uint64, 1024](1,2,3): limit 256 -> content depth 8, length mix-in 3.
func TestOracleListUint64(t *testing.T) {
	tree, root := traceHasher(t, func(h *Hasher) {
		h.PutUint64Array([]uint64{1, 2, 3}, 1024)
	})
	const want = "7d71cb79deb3cc392afd800f19c07b5733b177b0bcd92f607052a1ffe314efb0"
	if got := hex.EncodeToString(root[:]); got != want {
		t.Fatalf("List[uint64,1024](1,2,3) root:\n got %s\nwant %s (remerkleable oracle)", got, want)
	}
	_ = tree
}

// List[Point, 4]( Point(1,2), Point(3,4) ): composite elements merkleize
// element roots (not packed bytes); limit 4 -> content depth 2, length mix-in 2.
func TestOracleListOfComposite(t *testing.T) {
	_, root := traceHasher(t, func(h *Hasher) {
		indx := h.Index()
		e0 := h.Index()
		h.PutUint64(1)
		h.PutUint64(2)
		h.Merkleize(e0)
		e1 := h.Index()
		h.PutUint64(3)
		h.PutUint64(4)
		h.Merkleize(e1)
		h.MerkleizeWithMixin(indx, 2, 4)
	})
	const want = "e3f3d6d0bad233531bdde28f566bc73b449291e7a1ce9d2ef4c1cc2aba5df664"
	if got := hex.EncodeToString(root[:]); got != want {
		t.Fatalf("List[Point,4] root:\n got %s\nwant %s (remerkleable oracle)", got, want)
	}
}
