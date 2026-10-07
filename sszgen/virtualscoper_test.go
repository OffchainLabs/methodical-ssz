package sszgen

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/render"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

func TestNewVirtualPathScoper(t *testing.T) {
	cbytes, err := os.ReadFile("testdata/uint256.go.fixture")
	if err != nil {
		t.Fatal(err)
	}
	vf := VirtualFile{
		name:     "uint256.go",
		contents: string(cbytes),
	}
	pkgName := "github.com/ethereum/go-ethereum/faketypes"
	vps, err := NewVirtualPathScoper(pkgName, vf)
	if err != nil {
		t.Fatal(err)
	}
	if vps.Path() != pkgName {
		t.Fatalf("unexpected path, want=%s, got=%s", pkgName, vps.Path())
	}
	defs, err := TypeDefs(vps, "TestStruct")
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 {
		t.Fatalf("unexpected number of definitions, want=1, got=%d", len(defs))
	}
	typeRep, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatal(err)
	}
	rb, err := render.Render(pkgName, "", []gentypes.ValRep{typeRep})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/uint256.ssz.go.fixture")
	if err != nil {
		t.Fatal(err)
	}
	// compare string representations so test failures will be legible
	if string(expected) != string(rb) {
		t.Fatalf("unexpected result from generator render, want=%s, got=%s", string(expected), string(rb))
	}
}
