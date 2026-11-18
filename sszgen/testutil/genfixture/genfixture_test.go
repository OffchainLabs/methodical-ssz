package genfixture

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/ssz"
)

func TestBitsRoundTrip(t *testing.T) {
	in := Bits{V4: []byte{0x05}, V16: []byte{0xff, 0x01}}
	want := []byte{0x05, 0xff, 0x01}
	enc, err := in.MarshalSSZ()
	if err != nil || !bytes.Equal(enc, want) {
		t.Fatalf("MarshalSSZ() = %x, %v; want %x", enc, err, want)
	}
	var dec Bits
	if err := dec.UnmarshalSSZ(enc); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec.V4, in.V4) || !bytes.Equal(dec.V16, in.V16) {
		t.Fatalf("UnmarshalSSZ(%x) = %+v, want %+v", enc, dec, in)
	}
}

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
			in := Bits{V4: []byte{tt.v4}, V16: tt.v16[:]}
			got, err := in.MarshalSSZ()
			if tt.valid != (err == nil) {
				t.Fatalf("MarshalSSZ(%x) = %v, want valid=%v", enc, err, tt.valid)
			}
			if tt.valid && !bytes.Equal(got, enc) {
				t.Fatalf("MarshalSSZ = %x, want %x", got, enc)
			}
			if _, err := in.HashTreeRoot(); tt.valid != (err == nil) {
				t.Fatalf("HashTreeRoot(%x) = %v, want valid=%v", enc, err, tt.valid)
			}
		})
	}
}

func TestBitvector2Padding(t *testing.T) {
	for _, tt := range []struct {
		b     byte
		valid bool
	}{{0x03, true}, {0x04, false}, {0x80, false}} {
		var dec Bits2
		if err := dec.UnmarshalSSZ([]byte{tt.b}); tt.valid != (err == nil) {
			t.Fatalf("UnmarshalSSZ(%x) = %v, want valid=%v", tt.b, err, tt.valid)
		}
		in := Bits2{V2: []byte{tt.b}}
		if _, err := in.MarshalSSZ(); tt.valid != (err == nil) {
			t.Fatalf("MarshalSSZ(%x) = %v, want valid=%v", tt.b, err, tt.valid)
		}
		if _, err := in.HashTreeRoot(); tt.valid != (err == nil) {
			t.Fatalf("HashTreeRoot(%x) = %v, want valid=%v", tt.b, err, tt.valid)
		}
	}
}

func TestBitlistLimit(t *testing.T) {
	bitlist := func(n int, last byte) []byte {
		b := make([]byte, n)
		b[n-1] = last
		return b
	}
	tests := []struct {
		name  string
		l     []byte
		valid bool
	}{
		{"0 bits", []byte{0x01}, true},
		{"2048 bits", bitlist(257, 0x01), true},
		{"2049 bits", bitlist(257, 0x02), false},
		{"258 bytes", bitlist(258, 0x01), false},
		{"empty", []byte{}, false},
		{"last byte zero", []byte{0x01, 0x00}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := append([]byte{4, 0, 0, 0}, tt.l...) // offset 4, then L
			var dec ListBits
			if err := dec.UnmarshalSSZ(enc); tt.valid != (err == nil) {
				t.Fatalf("UnmarshalSSZ(%x) = %v, want valid=%v", enc, err, tt.valid)
			}
			got, err := (&ListBits{L: tt.l}).MarshalSSZ()
			if tt.valid != (err == nil) {
				t.Fatalf("MarshalSSZ() = %v, want valid=%v", err, tt.valid)
			}
			if tt.valid && (!bytes.Equal(got, enc) || !bytes.Equal(dec.L, tt.l)) {
				t.Fatalf("round trip: MarshalSSZ() = %x, decoded L = %x; want %x", got, dec.L, enc)
			}
			if _, err := (&ListBits{L: tt.l}).HashTreeRoot(); tt.valid != (err == nil) {
				t.Fatalf("HashTreeRoot() = %v, want valid=%v", err, tt.valid)
			}
		})
	}
}

func TestProgressiveBitlist(t *testing.T) {
	tests := []struct {
		name  string
		l     []byte
		valid bool
	}{
		{"0 bits", []byte{0x01}, true},
		{"8 bits, ends at a byte boundary", []byte{0xff, 0x01}, true},
		{"empty", []byte{}, false},
		{"one zero byte", []byte{0x00}, false},
		{"last byte zero", []byte{0x05, 0x00}, false},
		{"33 bytes, last byte zero", append(bytes.Repeat([]byte{0xff}, 32), 0x00), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := ProgBits{L: tt.l}
			if _, err := in.MarshalSSZ(); tt.valid != (err == nil) {
				t.Fatalf("MarshalSSZ() = %v, want valid=%v", err, tt.valid)
			}
			if _, err := in.HashTreeRoot(); tt.valid != (err == nil) {
				t.Fatalf("HashTreeRoot() = %v, want valid=%v", err, tt.valid)
			}
		})
	}
}

func TestListU64FirstOffset(t *testing.T) {
	enc := binary.LittleEndian.AppendUint64(nil, 7)
	enc = binary.LittleEndian.AppendUint32(enc, 12)
	enc = binary.LittleEndian.AppendUint64(enc, 9)
	var dec ListU64
	if err := dec.UnmarshalSSZ(enc); err != nil || dec.A != 7 || len(dec.L) != 1 || dec.L[0] != 9 {
		t.Fatalf("UnmarshalSSZ(%x) = %+v, %v", enc, dec, err)
	}
	// The fixed part is 8 + 4 = 12 bytes, so the first offset must be 12.
	for _, off := range []uint32{0, 11, 13, 20, 0xffffffff} {
		bad := append([]byte{}, enc...)
		binary.LittleEndian.PutUint32(bad[8:12], off)
		if err := dec.UnmarshalSSZ(bad); !errors.Is(err, ssz.ErrInvalidVariableOffset) {
			t.Fatalf("offset %d: UnmarshalSSZ = %v, want ErrInvalidVariableOffset", off, err)
		}
	}
}

// A List[byte, 32] of 33 bytes must fail to decode, encode and hash.
func TestByteListLimit(t *testing.T) {
	encodeExtra := func(n int) []byte {
		return append([]byte{4, 0, 0, 0}, bytes.Repeat([]byte{0xab}, n)...) // offset 4, then ExtraData
	}
	var dec Extra
	if err := dec.UnmarshalSSZ(encodeExtra(32)); err != nil || len(dec.ExtraData) != 32 {
		t.Fatalf("UnmarshalSSZ(32 bytes) = %v, len %d; want nil, 32", err, len(dec.ExtraData))
	}
	if err := dec.UnmarshalSSZ(encodeExtra(33)); !errors.Is(err, ssz.ErrListTooBig) {
		t.Fatalf("UnmarshalSSZ(33 bytes) = %v, want ErrListTooBig", err)
	}
	in := Extra{ExtraData: bytes.Repeat([]byte{0xab}, 33)}
	if _, err := in.MarshalSSZ(); err == nil {
		t.Fatal("MarshalSSZ accepted 33 bytes")
	}
	if _, err := in.HashTreeRoot(); err == nil {
		t.Fatal("HashTreeRoot accepted 33 bytes")
	}
}
