package genfixture

import (
	"bytes"
	"testing"
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
	}
}
