package sszgen

import (
	"testing"
)

func TestReformatStructTags(t *testing.T) {
	decl := `PublicKey                  []byte           "protobuf:\"bytes,1,opt,name=public_key,json=publicKey,proto3\" json:\"public_key,omitempty\" spec-name:\"pubkey\" ssz-size:\"48\""`
	// unquoted quotation marks should be converted to backticks
	expected := "PublicKey                  []byte `protobuf:\"bytes,1,opt,name=public_key,json=publicKey,proto3\" json:\"public_key,omitempty\" spec-name:\"pubkey\" ssz-size:\"48\"`"
	got := reformatStructTag(decl)
	if got != expected {
		t.Fatalf("unexpected result from reformatStructTag, want=%q, got=%q", expected, got)
	}
}
