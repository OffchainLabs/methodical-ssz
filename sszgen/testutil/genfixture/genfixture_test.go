package genfixture

import (
	"bytes"
	"testing"
)

// F1. Expected values come from the consensus-specs BitVector[N] serializer
// (ssz/simple-serialize.md): value[i] goes to bit i%8 of byte i//8 for i < N,
// so bits N.. of the last byte are always zero. EthLib's deserialize returns
// .paddingBits for a set padding bit. V4 has 4 padding bits; V16 has none.
func TestBitvectorPadding(t *testing.T) {
	tests := []struct {
		name  string
		v4    byte
		v16   [2]byte
		valid bool
	}{
		{"V4 value bits only", 0x0f, [2]byte{0xff, 0xff}, true},
		{"V4 padding bit 4 set", 0x10, [2]byte{}, false},
		{"V4 padding bit 7 set", 0x80, [2]byte{}, false},
		{"V16 all bits set", 0x00, [2]byte{0xff, 0xff}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := []byte{tt.v4, tt.v16[0], tt.v16[1]}
			var dec Bits
			if err := dec.UnmarshalSSZ(enc); tt.valid != (err == nil) {
				t.Fatalf("UnmarshalSSZ(%x) = %v, want valid=%v", enc, err, tt.valid)
			}
			// the encoder rejects the same values, so it never emits bytes
			// its own decoder rejects
			in := Bits{V4: []byte{tt.v4}, V16: tt.v16[:]}
			got, err := in.MarshalSSZ()
			if tt.valid != (err == nil) {
				t.Fatalf("MarshalSSZ(%x) = %v, want valid=%v", enc, err, tt.valid)
			}
			if tt.valid && !bytes.Equal(got, enc) {
				t.Fatalf("MarshalSSZ = %x, want %x", got, enc)
			}
		})
	}
}
