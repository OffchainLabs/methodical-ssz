package genfixture

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/ssz"
)

// encodeBits is the spec serialization of Bits: the fixed part (V4, V16, then
// a 4-byte little-endian offset to L, equal to the fixed size 1+2+4 = 7)
// followed by L's bytes.
func encodeBits(v4 byte, v16 [2]byte, l []byte) []byte {
	enc := []byte{v4, v16[0], v16[1]}
	enc = binary.LittleEndian.AppendUint32(enc, 7)
	return append(enc, l...)
}

// emptyBitlist is the spec serialization of an empty BitList: only the
// delimiter bit at index 0.
var emptyBitlist = []byte{0x01}

// checkBits asserts that both the decoder and the encoder accept (valid) or
// reject the value, and that an accepted value round-trips byte for byte.
func checkBits(t *testing.T, in Bits, valid bool) {
	t.Helper()
	enc := encodeBits(in.V4[0], [2]byte{in.V16[0], in.V16[1]}, in.L)
	var dec Bits
	if err := dec.UnmarshalSSZ(enc); valid != (err == nil) {
		t.Fatalf("UnmarshalSSZ(%x) = %v, want valid=%v", enc, err, valid)
	}
	if valid {
		// the decoded value must be the input, and must re-encode to enc
		if !bytes.Equal(dec.V4, in.V4) || !bytes.Equal(dec.V16, in.V16) || !bytes.Equal(dec.L, in.L) {
			t.Fatalf("UnmarshalSSZ(%x) = %+v, want %+v", enc, dec, in)
		}
		if re, err := dec.MarshalSSZ(); err != nil || !bytes.Equal(re, enc) {
			t.Fatalf("decoded MarshalSSZ() = %x, %v; want %x", re, err, enc)
		}
	}
	// the encoder rejects the same values, so it never emits bytes its own
	// decoder rejects
	got, err := in.MarshalSSZ()
	if valid != (err == nil) {
		t.Fatalf("MarshalSSZ() = %v, want valid=%v", err, valid)
	}
	if valid && !bytes.Equal(got, enc) {
		t.Fatalf("MarshalSSZ() = %x, want %x", got, enc)
	}
}

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
			checkBits(t, Bits{V4: []byte{tt.v4}, V16: tt.v16[:], L: emptyBitlist}, tt.valid)
		})
	}
}

// bitlistBytes is a Go bitlist of n bytes whose last byte is last and the
// rest zero.
func bitlistBytes(n int, last byte) []byte {
	b := make([]byte, n)
	b[n-1] = last
	return b
}

// F2. Expected values come from the consensus-specs BitList[N] serializer
// (ssz/simple-serialize.md): a bitlist of length e <= N serializes to e//8+1
// bytes with the delimiter bit at index e, so the last byte is never zero,
// the encoding is never empty, and e = 8*(bytes-1) + msb(last) - 1 <= N.
// For N = 2048 the longest valid encoding is 257 bytes with last byte 0x01.
func TestBitlistLimit(t *testing.T) {
	tests := []struct {
		name  string
		l     []byte
		valid bool
	}{
		{"empty byte array", []byte{}, false},
		{"last byte zero", []byte{0x01, 0x00}, false},
		{"2048 bits (257 bytes, last 0x01)", bitlistBytes(257, 0x01), true},
		{"2049 bits (257 bytes, last 0x02)", bitlistBytes(257, 0x02), false},
		{"258 bytes (2056 bits)", bitlistBytes(258, 0x01), false},
		{"0 bits", emptyBitlist, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBits(t, Bits{V4: []byte{0}, V16: []byte{0, 0}, L: tt.l}, tt.valid)
		})
	}
}

// G1. A [4]uint64 field decodes into the array in place. Expected values
// from consensus-specs: Vector[uint64, 4] serializes as 4 little-endian
// uint64s (32 bytes). Its root is pack(A), one 32-byte chunk, and the root of
// a one-field container with a one-chunk field is that chunk, so the root of
// Limbs equals its serialization.
func TestUint64Array(t *testing.T) {
	in := Limbs{A: [4]uint64{1, 2, 3, 0xffffffffffffffff}}
	want := []byte{}
	for _, x := range in.A {
		want = binary.LittleEndian.AppendUint64(want, x)
	}
	enc, err := in.MarshalSSZ()
	if err != nil || !bytes.Equal(enc, want) {
		t.Fatalf("MarshalSSZ() = %x, %v; want %x", enc, err, want)
	}
	var dec Limbs
	if err := dec.UnmarshalSSZ(enc); err != nil || dec != in {
		t.Fatalf("UnmarshalSSZ = %v, %v; want %v", dec, err, in)
	}
	root, err := in.HashTreeRoot()
	if err != nil || !bytes.Equal(root[:], want) {
		t.Fatalf("HashTreeRoot() = %x, %v; want %x", root, err, want)
	}
}

// G3. The first offset of a container must equal the size of its fixed part
// (EthLib deserialize, remerkleable decode_bytes); here 8 + 4 = 12. Any
// other value is rejected, including one past the end of the input, so a
// separate "offset > size" check is unreachable.
func TestListU64FirstOffset(t *testing.T) {
	enc := binary.LittleEndian.AppendUint64(nil, 7)
	enc = binary.LittleEndian.AppendUint32(enc, 12)
	enc = binary.LittleEndian.AppendUint64(enc, 9)
	var dec ListU64
	if err := dec.UnmarshalSSZ(enc); err != nil || dec.A != 7 || len(dec.L) != 1 || dec.L[0] != 9 {
		t.Fatalf("UnmarshalSSZ(%x) = %+v, %v", enc, dec, err)
	}
	for _, off := range []uint32{0, 11, 13, 20, 0xffffffff} {
		bad := append([]byte{}, enc...)
		binary.LittleEndian.PutUint32(bad[8:12], off)
		if err := dec.UnmarshalSSZ(bad); !errors.Is(err, ssz.ErrInvalidVariableOffset) {
			t.Fatalf("offset %d: UnmarshalSSZ = %v, want ErrInvalidVariableOffset", off, err)
		}
	}
}

// G2. MarshalSSZTo writes A and L's offset before it checks L against its
// limit, so on error the bytes past len(dst) are unspecified (see the
// ssz.Marshaler doc). This pins what the doc does promise: a nil result and
// an unchanged dst[:len(dst)].
func TestMarshalSSZToErrorKeepsPrefix(t *testing.T) {
	in := ListU64{A: 1, L: make([]uint64, 5)} // 5 > ssz-max 4
	dst := bytes.Repeat([]byte{0xaa}, 64)[:3]
	got, err := in.MarshalSSZTo(dst)
	if !errors.Is(err, ssz.ErrListTooBig) || got != nil {
		t.Fatalf("MarshalSSZTo = %x, %v; want nil, ErrListTooBig", got, err)
	}
	if !bytes.Equal(dst, []byte{0xaa, 0xaa, 0xaa}) {
		t.Fatalf("dst[:len(dst)] changed: %x", dst)
	}
}

// G1, variable-size elements. Expected bytes from the consensus-specs
// serializer, built by hand: Inner{B} is offset 4 then B; Vector[Inner, 2]
// is two offsets (8, 8+6 = 14) then the two Inner encodings; InnerPair is
// offset 4 then P.
func TestVariableElementArray(t *testing.T) {
	in := InnerPair{P: [2]*Inner{{B: []byte{1, 2}}, {B: []byte{3}}}}
	want := []byte{
		4, 0, 0, 0, // InnerPair: offset of P
		8, 0, 0, 0, 14, 0, 0, 0, // P: element offsets
		4, 0, 0, 0, 1, 2, // P[0]
		4, 0, 0, 0, 3, // P[1]
	}
	enc, err := in.MarshalSSZ()
	if err != nil || !bytes.Equal(enc, want) {
		t.Fatalf("MarshalSSZ() = %x, %v; want %x", enc, err, want)
	}
	var dec InnerPair
	if err := dec.UnmarshalSSZ(want); err != nil {
		t.Fatal(err)
	}
	for i := range in.P {
		if dec.P[i] == nil || !bytes.Equal(dec.P[i].B, in.P[i].B) {
			t.Fatalf("UnmarshalSSZ P[%d] = %+v, want %+v", i, dec.P[i], in.P[i])
		}
	}
	// the first element offset must be 4*2 = 8
	bad := append([]byte{}, want...)
	bad[4] = 4
	if err := dec.UnmarshalSSZ(bad); err == nil {
		t.Fatal("UnmarshalSSZ accepted a first element offset of 4")
	}
}
