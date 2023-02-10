package sszgen

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestReformatStructTags(t *testing.T) {
	decl := `PublicKey                  []byte           "protobuf:\"bytes,1,opt,name=public_key,json=publicKey,proto3\" json:\"public_key,omitempty\" spec-name:\"pubkey\" ssz-size:\"48\""`
	// unquoted quotation marks should be converted to backticks
	expected := "PublicKey                  []byte `protobuf:\"bytes,1,opt,name=public_key,json=publicKey,proto3\" json:\"public_key,omitempty\" spec-name:\"pubkey\" ssz-size:\"48\"`"
	got := reformatStructTag(decl)
	require.Equal(t, expected, got)
}
