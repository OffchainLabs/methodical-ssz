package ssz

import (
	"encoding/binary"
	"fmt"
	"testing"

	sha256 "github.com/minio/sha256-simd"
)

// Naive reference implementations following ssz-spec.md's text directly,
// independent of the optimized production code (gohashtree, zero-hash caches,
// in-place scratch). The production results must match these.

func refHash(a, b [32]byte) [32]byte {
	return sha256.Sum256(append(a[:], b[:]...))
}

// refMerkleize: binary merkleization of chunks zero-padded to limit leaves.
func refMerkleize(chunks [][32]byte, limit uint64) [32]byte {
	width := uint64(1)
	for width < limit {
		width *= 2
	}
	layer := make([][32]byte, width)
	copy(layer, chunks)
	for len(layer) > 1 {
		next := make([][32]byte, len(layer)/2)
		for i := range next {
			next[i] = refHash(layer[2*i], layer[2*i+1])
		}
		layer = next
	}
	return layer[0]
}

// refProgressive: merkleize_progressive(chunks, num_leaves) per the spec —
// hash(merkleize(chunks[:n], numLeaves), merkleize_progressive(chunks[n:], numLeaves*4)).
func refProgressive(chunks [][32]byte, numLeaves uint64) [32]byte {
	if len(chunks) == 0 {
		return [32]byte{}
	}
	n := numLeaves
	if uint64(len(chunks)) < n {
		n = uint64(len(chunks))
	}
	a := refMerkleize(chunks[:n], numLeaves)
	b := refProgressive(chunks[n:], numLeaves*4)
	return refHash(a, b)
}

// testChunks builds count distinct chunks.
func testChunks(count int) [][32]byte {
	chunks := make([][32]byte, count)
	for i := range chunks {
		binary.LittleEndian.PutUint64(chunks[i][:8], uint64(i+1))
		chunks[i][31] = 0xAA
	}
	return chunks
}

// Counts straddle the progressive subtree boundaries (capacities 1, 4, 16, 64
// cumulate to 1, 5, 21, 85).
var progressiveCounts = []int{0, 1, 2, 4, 5, 6, 20, 21, 22, 84, 85, 86, 200}

func TestMerkleizeProgressiveAgainstReference(t *testing.T) {
	for _, count := range progressiveCounts {
		t.Run(fmt.Sprintf("%d_chunks", count), func(t *testing.T) {
			want := refProgressive(testChunks(count), 1)
			// production code consumes its input as scratch: pass a copy
			got := merkleizeProgressive(testChunks(count), 1)
			if got != want {
				t.Fatalf("merkleizeProgressive mismatch for %d chunks:\nwant %x\ngot  %x", count, want, got)
			}
		})
	}
}

func TestHasherMerkleizeProgressive(t *testing.T) {
	for _, count := range progressiveCounts {
		t.Run(fmt.Sprintf("%d_chunks", count), func(t *testing.T) {
			hh := DefaultHasherPool.Get()
			defer DefaultHasherPool.Put(hh)
			indx := hh.Index()
			for _, c := range testChunks(count) {
				hh.Append(c[:])
			}
			hh.MerkleizeProgressive(indx)
			got, err := hh.HashRoot()
			if err != nil {
				t.Fatal(err)
			}
			if want := refProgressive(testChunks(count), 1); got != want {
				t.Fatalf("want %x got %x", want, got)
			}
		})
	}
}

func TestMerkleizeProgressiveWithMixin(t *testing.T) {
	const count = 6
	hh := DefaultHasherPool.Get()
	defer DefaultHasherPool.Put(hh)
	indx := hh.Index()
	for _, c := range testChunks(count) {
		hh.Append(c[:])
	}
	hh.MerkleizeProgressiveWithMixin(indx, uint64(count))
	got, err := hh.HashRoot()
	if err != nil {
		t.Fatal(err)
	}
	var lenChunk [32]byte
	binary.LittleEndian.PutUint64(lenChunk[:8], uint64(count))
	if want := refHash(refProgressive(testChunks(count), 1), lenChunk); got != want {
		t.Fatalf("want %x got %x", want, got)
	}
}

func TestMerkleizeProgressiveWithActiveFields(t *testing.T) {
	const count = 3
	activeFields := []byte{0b00001011} // fields at positions 0, 1, 3
	hh := DefaultHasherPool.Get()
	defer DefaultHasherPool.Put(hh)
	indx := hh.Index()
	for _, c := range testChunks(count) {
		hh.Append(c[:])
	}
	hh.MerkleizeProgressiveWithActiveFields(indx, activeFields)
	got, err := hh.HashRoot()
	if err != nil {
		t.Fatal(err)
	}
	var afChunk [32]byte
	copy(afChunk[:], activeFields)
	if want := refHash(refProgressive(testChunks(count), 1), afChunk); got != want {
		t.Fatalf("want %x got %x", want, got)
	}
}

func TestPutProgressiveBitlist(t *testing.T) {
	// 0b00000111: two data bits set (positions 0 and 1), delimiter at bit 2.
	bb := []byte{0b00000111}
	hh := DefaultHasherPool.Get()
	defer DefaultHasherPool.Put(hh)
	hh.PutProgressiveBitlist(bb)
	got, err := hh.HashRoot()
	if err != nil {
		t.Fatal(err)
	}
	var bits [32]byte
	bits[0] = 0b00000011
	var lenChunk [32]byte
	binary.LittleEndian.PutUint64(lenChunk[:8], 2)
	if want := refHash(refProgressive([][32]byte{bits}, 1), lenChunk); got != want {
		t.Fatalf("want %x got %x", want, got)
	}
}

// serializeBitlist packs bits little-endian and appends the length-delimiting
// bit, producing the SSZ wire encoding of a bitlist (test input construction).
func serializeBitlist(bits []bool) []byte {
	buf := make([]byte, len(bits)/8+1)
	for i, b := range bits {
		if b {
			buf[i/8] |= 1 << (uint(i) % 8)
		}
	}
	buf[len(bits)/8] |= 1 << (uint(len(bits)) % 8) // delimiter
	return buf
}

// refPackBits packs bits into ceil(len/256) chunks — the chunk count is implied
// by the bit length, so a fully-zero final chunk is retained.
func refPackBits(bits []bool) [][32]byte {
	chunks := make([][32]byte, (len(bits)+255)/256)
	for i, b := range bits {
		if b {
			chunks[i/256][(i%256)/8] |= 1 << (uint(i) % 8)
		}
	}
	return chunks
}

// TestPutProgressiveBitlistTrailingZeroChunks guards the regression where a
// progressive bitlist whose final 256-bit chunk is entirely zero was merkleized
// with that chunk dropped (parseBitlist trims trailing zero bytes; for a bounded
// bitlist MerkleizeWithMixin re-pads to the limit, but a progressive bitlist has
// no limit). The "single_zero_bit" and "trailing_zero_chunk_257" cases fail on
// the trimming implementation and pass on the fixed one.
func TestPutProgressiveBitlistTrailingZeroChunks(t *testing.T) {
	zeros := func(n int, set ...int) []bool {
		b := make([]bool, n)
		for _, i := range set {
			b[i] = true
		}
		return b
	}
	cases := map[string][]bool{
		"empty":                   {},
		"single_zero_bit":         {false},        // 1 chunk, all zero (would trim to 0 chunks)
		"single_set_bit":          {true},         // sanity
		"two_bits_10":             {true, false},  // sanity
		"trailing_zero_chunk_257": zeros(257, 0),  // 2 chunks, 2nd all zero (would trim to 1 chunk)
	}
	for name, bits := range cases {
		t.Run(name, func(t *testing.T) {
			hh := DefaultHasherPool.Get()
			defer DefaultHasherPool.Put(hh)
			hh.PutProgressiveBitlist(serializeBitlist(bits))
			got, err := hh.HashRoot()
			if err != nil {
				t.Fatal(err)
			}
			var lenChunk [32]byte
			binary.LittleEndian.PutUint64(lenChunk[:8], uint64(len(bits)))
			if want := refHash(refProgressive(refPackBits(bits), 1), lenChunk); got != want {
				t.Fatalf("want %x got %x", want, got)
			}
		})
	}
}
