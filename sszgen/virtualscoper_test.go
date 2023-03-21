package sszgen

import (
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/backend"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestNewVirtualPathScoper(t *testing.T) {
	cbytes, err := os.ReadFile("testdata/uint256.go.fixture")
	require.NoError(t, err)
	vf := VirtualFile{
		name:     "uint256.go",
		contents: string(cbytes),
	}
	expected, err := os.ReadFile("testdata/uint256.ssz.go.fixture")
	require.NoError(t, err)
	pkgName := "github.com/ethereum/go-ethereum/faketypes"
	vps, err := NewVirtualPathScoper(pkgName, vf)
	require.NoError(t, err)
	require.Equal(t, pkgName, vps.Path())
	defs, err := TypeDefs(vps, "TestStruct")
	require.NoError(t, err)
	require.Equal(t, 1, len(defs))
	typeRep, err := ParseTypeDef(defs[0])
	require.NoError(t, err)
	g := backend.NewGenerator(pkgName)
	require.NoError(t, g.Generate(typeRep))
	rb, err := g.Render()
	require.NoError(t, err)
	// compare string representations so test failures will be legible
	require.Equal(t, string(expected), string(rb))
}
