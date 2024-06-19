package specs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestDecodeRootFile(t *testing.T) {
	expectedHex := "0x44de62c118d7951f5b6d9a03444e54aff47d02ff57add2a4eb2a198b3e83ae35"
	e, err := hexutil.Decode(expectedHex)
	if err != nil {
		t.Fatalf("hexutil.Decode failed on input \"%s\"", expectedHex)
	}
	expected := [32]byte{}
	copy(expected[:], e)
	f := []byte(`{root: '0x44de62c118d7951f5b6d9a03444e54aff47d02ff57add2a4eb2a198b3e83ae35'}`)
	r, err := DecodeRootFile(f)
	if err != nil {
		t.Fatalf("unexpected error result from DecodeRootFile, %s", err.Error())
	}
	if expected != r {
		t.Fatalf("root return value %#x from DecodeRootFile did not match expected value %#x", r, expected)
	}
}
