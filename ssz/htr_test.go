package ssz

import (
	"bytes"
	"crypto/sha256"
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/gohashtree"
)

func TestNextPowerOfTwo(t *testing.T) {
	cases := []struct {
		Num, Res uint64
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{6, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{10, 16},
		{11, 16},
		{13, 16},
	}
	for _, c := range cases {
		if next := nextPowerOfTwo(c.Num); uint64(next) != c.Res {
			t.Fatalf("num %d, expected %d but found %d", c.Num, c.Res, next)
		}
	}
}

func TestMerkleize8ByteVector(t *testing.T) {
	result := merkleizeInput([]byte{'1', '2', '3', '4', '5', '6', '7', '8'}, 0)
	if !bytes.Equal(result, []byte{49, 50, 51, 52, 53, 54, 55, 56, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("Unexpected result: %v", result)
	}
}

// H1. merkleizeChunks calls gohashtree.Hash(elements, elements) in place. This
// compares that call with a non-aliased call and with the spec definition
// (consensus-specs merkleize: digest i = SHA-256(chunk 2i || chunk 2i+1)),
// for 1..40 pairs so that every batch size (1, 2, 4, 8, 16) and its
// remainders occur. Only the path the host CPU selects runs. On an Apple M5
// Pro, native arm64 runs the arm64 sha2 path, and GOARCH=amd64 under Rosetta
// (no AVX2, no SHA reported) runs the generic Go path. The arm64 neon path
// and the amd64 avx1/avx2/avx512/shani paths run only on hosts that select
// them.
func TestMerkleizeInPlaceHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for pairs := 1; pairs <= 40; pairs++ {
		chunks := make([][32]byte, 2*pairs)
		for i := range chunks {
			rng.Read(chunks[i][:])
		}
		want := make([][32]byte, pairs)
		for i := range want {
			want[i] = sha256.Sum256(append(chunks[2*i][:], chunks[2*i+1][:]...))
		}
		separate := make([][32]byte, pairs)
		if err := gohashtree.Hash(separate, append([][32]byte{}, chunks...)); err != nil {
			t.Fatal(err)
		}
		inPlace := append([][32]byte{}, chunks...)
		if err := gohashtree.Hash(inPlace, inPlace); err != nil {
			t.Fatal(err)
		}
		for i := range want {
			if separate[i] != want[i] || inPlace[i] != want[i] {
				t.Fatalf("pairs=%d digest %d: separate %x, in place %x, want %x", pairs, i, separate[i], inPlace[i], want[i])
			}
		}
	}
}
