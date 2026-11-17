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
