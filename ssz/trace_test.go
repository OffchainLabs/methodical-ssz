package ssz

import (
	"encoding/hex"
	"testing"
)

// TestTraceRecordsContainerMerkleize checks the P1 instrumentation: a two-field
// container (Container{a:uint64=1, b:uint64=2}) drives two PutUint64 + one
// Merkleize, which must record exactly one balanced-merkleize event whose input
// is the two field chunks, and whose resulting root matches the design-doc value.
func TestTraceRecordsContainerMerkleize(t *testing.T) {
	h := NewHasher()
	h.EnableTrace()

	indx := h.Index()
	h.PutUint64(1)
	h.PutUint64(2)
	h.Merkleize(indx)

	root, err := h.HashRoot()
	if err != nil {
		t.Fatal(err)
	}
	const want = "ff55c97976a840b4ced964ed49e3794594ba3f675238b5fd25d282b60f70a194"
	if got := hex.EncodeToString(root[:]); got != want {
		t.Fatalf("root: want %s got %s", want, got)
	}

	evs := h.TraceEvents()
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %d", len(evs))
	}
	e := evs[0]
	if e.op != opMerkleize {
		t.Fatalf("want opMerkleize, got %d", e.op)
	}
	if e.bufStart != 0 {
		t.Fatalf("want bufStart 0, got %d", e.bufStart)
	}
	if len(e.input) != 64 {
		t.Fatalf("want 64-byte (2-chunk) input, got %d bytes", len(e.input))
	}
	// first chunk = uint64(1) LE, second = uint64(2) LE
	if hex.EncodeToString(e.input[0:8]) != "0100000000000000" || hex.EncodeToString(e.input[32:40]) != "0200000000000000" {
		t.Fatalf("unexpected input chunks: %x", e.input)
	}
}

// TestTraceProgressiveContainerActiveFields checks a progressive container path:
// a two-field progressive container records a MerkleizeProgressiveWithActiveFields
// event carrying the packed active-fields bitvector.
func TestTraceProgressiveContainerActiveFields(t *testing.T) {
	h := NewHasher()
	h.EnableTrace()

	indx := h.Index()
	h.PutUint64(1)
	h.PutUint64(2)
	h.MerkleizeProgressiveWithActiveFields(indx, []byte{0x03}) // both fields active

	if _, err := h.HashRoot(); err != nil {
		t.Fatal(err)
	}
	evs := h.TraceEvents()
	if len(evs) != 1 || evs[0].op != opProgressiveActiveFields {
		t.Fatalf("want 1 progressive-active-fields event, got %+v", evs)
	}
	if len(evs[0].activeFields) != 1 || evs[0].activeFields[0] != 0x03 {
		t.Fatalf("active fields not captured: %x", evs[0].activeFields)
	}
	if len(evs[0].input) != 64 {
		t.Fatalf("want 64-byte input, got %d", len(evs[0].input))
	}
}

// TestTraceDisabledZeroOverhead confirms no events are recorded when tracing is off.
func TestTraceDisabledZeroOverhead(t *testing.T) {
	h := NewHasher()
	indx := h.Index()
	h.PutUint64(1)
	h.PutUint64(2)
	h.Merkleize(indx)
	if _, err := h.HashRoot(); err != nil {
		t.Fatal(err)
	}
	if h.TracingEnabled() {
		t.Fatal("tracing should be off")
	}
	if evs := h.TraceEvents(); evs != nil {
		t.Fatalf("expected nil events with tracing off, got %d", len(evs))
	}
}
