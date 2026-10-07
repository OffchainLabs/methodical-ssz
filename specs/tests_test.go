package specs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

func TestExtractTarballCasesRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	name := "tests/mainnet/phase0/ssz_static/Fork/ssz_random/case_0/../../../../../../../../x/roots.yaml"
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: 0}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractTarballCases(&buf, TestIdent{}); err == nil {
		t.Fatalf("expected error for path traversal entry %q", name)
	}
}
