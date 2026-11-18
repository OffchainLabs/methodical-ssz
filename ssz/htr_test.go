package ssz

import (
	"bytes"
	"testing"
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

func TestPutBitlistInvalid(t *testing.T) {
	tests := []struct {
		name  string
		put   func(h *Hasher)
		valid bool
	}{
		{"0 bits", func(h *Hasher) { h.PutBitlist([]byte{0x01}, 2048) }, true},
		{"2048 bits", func(h *Hasher) { h.PutBitlist(append(make([]byte, 256), 0x01), 2048) }, true},
		{"empty", func(h *Hasher) { h.PutBitlist(nil, 2048) }, false},
		{"last byte zero", func(h *Hasher) { h.PutBitlist([]byte{0x01, 0x00}, 2048) }, false},
		{"2049 bits", func(h *Hasher) { h.PutBitlist(append(make([]byte, 256), 0x02), 2048) }, false},
		{"258 bytes", func(h *Hasher) { h.PutBitlist(append(make([]byte, 257), 0x01), 2048) }, false},
		{"progressive 0 bits", func(h *Hasher) { h.PutProgressiveBitlist([]byte{0x01}) }, true},
		{"progressive 8 bits", func(h *Hasher) { h.PutProgressiveBitlist([]byte{0xff, 0x01}) }, true},
		{"progressive empty", func(h *Hasher) { h.PutProgressiveBitlist(nil) }, false},
		{"progressive one zero byte", func(h *Hasher) { h.PutProgressiveBitlist([]byte{0x00}) }, false},
		{"progressive last byte zero", func(h *Hasher) { h.PutProgressiveBitlist([]byte{0x05, 0x00}) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHasher()
			tt.put(h)
			if _, err := h.HashRoot(); tt.valid != (err == nil) {
				t.Fatalf("HashRoot() = %v, want valid=%v", err, tt.valid)
			}
			// Reset clears the error, so that a pooled hasher can hash again.
			h.Reset()
			h.PutBitlist([]byte{0x01}, 8)
			if _, err := h.HashRoot(); err != nil {
				t.Fatalf("HashRoot() after Reset = %v", err)
			}
		})
	}
}
