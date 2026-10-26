package ssz

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// traceHasher runs build against a trace-enabled hasher, computes the root, and
// reconstructs + self-verifies the merkle tree. It fails the test on any error
// or self-verify mismatch and returns the root node and root hash.
func traceHasher(t *testing.T, build func(h *Hasher)) (*TraceNode, [32]byte) {
	t.Helper()
	h := NewHasher()
	h.EnableTrace()
	build(h)
	root, err := h.HashRoot()
	if err != nil {
		t.Fatalf("HashRoot: %v", err)
	}
	tree, err := reconstruct(h.TraceEvents())
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}
	if got := tree.verify(); got != root {
		t.Fatalf("self-verify mismatch: tree %x != root %x", got, root)
	}
	if tree.G != 1 {
		t.Fatalf("root G = %d, want 1", tree.G)
	}
	if tree.Root != root {
		t.Fatalf("root node root %x != HashRoot %x", tree.Root, root)
	}
	return tree, root
}

func le64Chunk(v uint64) [32]byte {
	var b [32]byte
	binary.LittleEndian.PutUint64(b[:8], v)
	return b
}

// chunkN builds a distinguishable 32-byte chunk (first 8 bytes = n).
func chunkN(n uint64) [32]byte { return le64Chunk(n) }

func TestReconstructContainer2(t *testing.T) {
	tree, _ := traceHasher(t, func(h *Hasher) {
		indx := h.Index()
		h.PutUint64(1)
		h.PutUint64(2)
		h.Merkleize(indx)
	})
	const want = "ff55c97976a840b4ced964ed49e3794594ba3f675238b5fd25d282b60f70a194"
	if got := hex.EncodeToString(tree.Root[:]); got != want {
		t.Fatalf("root: want %s got %s", want, got)
	}
	if tree.Left == nil || tree.Left.G != 2 || !tree.Left.Leaf || tree.Left.Root != le64Chunk(1) {
		t.Fatalf("g2 field a wrong: %+v", tree.Left)
	}
	if tree.Right == nil || tree.Right.G != 3 || !tree.Right.Leaf || tree.Right.Root != le64Chunk(2) {
		t.Fatalf("g3 field b wrong: %+v", tree.Right)
	}
}

// A 3-field container has depth 2; the 4th leaf slot (g7) is padding and, being a
// depth-0 zero, must render as a plain all-zero leaf (not a zero subtree).
func TestReconstructContainer3Padding(t *testing.T) {
	tree, _ := traceHasher(t, func(h *Hasher) {
		indx := h.Index()
		h.PutUint64(1)
		h.PutUint64(2)
		h.PutUint64(3)
		h.Merkleize(indx)
	})
	// g1 -> g2 (fields 0,1), g3 (field 2 + pad)
	g4 := tree.Left.Left
	g5 := tree.Left.Right
	g6 := tree.Right.Left
	g7 := tree.Right.Right
	if g4.G != 4 || g4.Root != le64Chunk(1) {
		t.Fatalf("g4 wrong: %+v", g4)
	}
	if g5.G != 5 || g5.Root != le64Chunk(2) {
		t.Fatalf("g5 wrong: %+v", g5)
	}
	if g6.G != 6 || g6.Root != le64Chunk(3) {
		t.Fatalf("g6 wrong: %+v", g6)
	}
	if g7.G != 7 || !g7.Leaf || g7.Zero || g7.Root != ([32]byte{}) {
		t.Fatalf("g7 pad should be all-zero leaf, got %+v", g7)
	}
}

// List[uint64, 1024] with 3 elements: MerkleizeWithMixin, limit 256 -> content
// depth 8. Content at g2 (one packed chunk at g512, zero subtrees elided along
// the right spine), length mix-in leaf = 3 at g3.
func TestReconstructListWithMixinAndElision(t *testing.T) {
	tree, _ := traceHasher(t, func(h *Hasher) {
		h.PutUint64Array([]uint64{1, 2, 3}, 1024)
	})
	if tree.Left.G != 2 {
		t.Fatalf("content G = %d, want 2", tree.Left.G)
	}
	if tree.Right.G != 3 || !tree.Right.Leaf || tree.Right.Root != le64Chunk(3) {
		t.Fatalf("g3 length mix-in wrong: %+v", tree.Right)
	}
	// content right sibling is a depth-7 zero subtree at g5.
	g5 := tree.Left.Right
	if g5.G != 5 || !g5.Zero || g5.ZeroDepth != 7 {
		t.Fatalf("g5 should be zero depth 7, got %+v", g5)
	}
	if g5.Root != zeroHashes[7] {
		t.Fatalf("g5 root != zeroHashes[7]")
	}
	// descend the left spine to the single packed data chunk at g512.
	n := tree.Left
	for n.G < 512 {
		n = n.Left
	}
	if n.G != 512 || !n.Leaf {
		t.Fatalf("expected packed leaf at g512, got G=%d leaf=%v", n.G, n.Leaf)
	}
	// packs uint64(1,2,3) = 24 bytes into one chunk.
	var wantChunk [32]byte
	binary.LittleEndian.PutUint64(wantChunk[0:], 1)
	binary.LittleEndian.PutUint64(wantChunk[8:], 2)
	binary.LittleEndian.PutUint64(wantChunk[16:], 3)
	if n.Root != wantChunk {
		t.Fatalf("g512 packed chunk wrong: got %x want %x", n.Root, wantChunk)
	}
}

// Progressive list of 6 chunks: windows of capacity 1,4,16 along a right-leaning
// spine. Verify the spine gindices emerge uniformly (window0 leaf at g4, window1
// leaves at g40-43) and the length mix-in is at g3.
func TestReconstructProgressiveMixinSpine(t *testing.T) {
	chunks := []([32]byte){chunkN(1), chunkN(2), chunkN(3), chunkN(4), chunkN(5), chunkN(6)}
	tree, _ := traceHasher(t, func(h *Hasher) {
		indx := h.Index()
		for _, c := range chunks {
			h.Append(c[:])
		}
		h.MerkleizeProgressiveWithMixin(indx, uint64(len(chunks)))
	})
	// g3 length mix-in = 6.
	if tree.Right.G != 3 || tree.Right.Root != le64Chunk(6) {
		t.Fatalf("g3 length wrong: %+v", tree.Right)
	}
	prog := tree.Left // g2 progressive spine root
	if prog.G != 2 {
		t.Fatalf("progressive root G = %d, want 2", prog.G)
	}
	// window 0 (capacity 1) is a single leaf at g4 == chunk 1.
	w0 := prog.Left
	if w0.G != 4 || !w0.Leaf || w0.Root != chunkN(1) {
		t.Fatalf("window0 wrong: %+v", w0)
	}
	// spine continues right at g5; window 1 root at g10, first leaf at g40.
	rest := prog.Right
	if rest.G != 5 {
		t.Fatalf("spine g5 = %d", rest.G)
	}
	w1 := rest.Left
	if w1.G != 10 {
		t.Fatalf("window1 root G = %d, want 10", w1.G)
	}
	w1first := w1.Left.Left // g20 -> g40
	if w1first.G != 40 || w1first.Root != chunkN(2) {
		t.Fatalf("window1 first leaf: G=%d root=%x", w1first.G, w1first.Root)
	}
}

// Progressive container: progressive spine at g2, active-fields mix-in at g3.
func TestReconstructProgressiveActiveFields(t *testing.T) {
	tree, _ := traceHasher(t, func(h *Hasher) {
		indx := h.Index()
		h.PutUint64(1)
		h.PutUint64(2)
		h.MerkleizeProgressiveWithActiveFields(indx, []byte{0x03})
	})
	if tree.Left.G != 2 {
		t.Fatalf("progressive content G = %d, want 2", tree.Left.G)
	}
	af := tree.Right
	if af.G != 3 || !af.Leaf || af.Root[0] != 0x03 {
		t.Fatalf("active-fields mix-in wrong: %+v", af)
	}
	// two fields -> window0 (cap1) = field0 at g4, spine right g5 holds field1.
	if tree.Left.Left.G != 4 || tree.Left.Left.Root != le64Chunk(1) {
		t.Fatalf("field0 at g4 wrong: %+v", tree.Left.Left)
	}
}

// A container whose first field is itself a sub-container exercises buffer
// containment with a shared bufStart (inner and outer both start at 0).
func TestReconstructNestedContainer(t *testing.T) {
	tree, _ := traceHasher(t, func(h *Hasher) {
		indx := h.Index()
		sub := h.Index()
		h.PutUint64(7)
		h.PutUint64(8)
		h.Merkleize(sub) // field 0: sub-container root at buf pos 0
		h.PutUint64(9)   // field 1
		h.Merkleize(indx)
	})
	// g2 is the sub-container (a branch), g3 is field 1 (leaf 9).
	if tree.Left.G != 2 || tree.Left.Leaf || tree.Left.Left == nil {
		t.Fatalf("g2 should be a branch (sub-container): %+v", tree.Left)
	}
	if tree.Left.Left.G != 4 || tree.Left.Left.Root != le64Chunk(7) {
		t.Fatalf("sub field x wrong: %+v", tree.Left.Left)
	}
	if tree.Left.Right.G != 5 || tree.Left.Right.Root != le64Chunk(8) {
		t.Fatalf("sub field y wrong: %+v", tree.Left.Right)
	}
	if tree.Right.G != 3 || !tree.Right.Leaf || tree.Right.Root != le64Chunk(9) {
		t.Fatalf("g3 field 1 wrong: %+v", tree.Right)
	}
}
