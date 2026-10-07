package sszgen

import (
	"testing"
)

func TestTokens(t *testing.T) {
	testTag := "`protobuf:\"bytes,2004,rep,name=historical_roots,json=historicalRoots,proto3\" json:\"historical_roots,omitempty\" ssz-max:\"16777216\" ssz-size:\"?,32\"`"
	tp := &TagParser{}
	tp.Init(testTag)
	tags := tp.GetSSZTags()
	sszSize, ok := tags["ssz-size"]
	if !ok {
		t.Fatalf("ssz-size not found")
	}
	if sszSize != "?,32" {
		t.Fatalf("ssz-size not correct, expected ?,32, got %s", sszSize)
	}
	sszMax, ok := tags["ssz-max"]
	if !ok {
		t.Fatalf("ssz-max not found")
	}
	if sszMax != "16777216" {
		t.Fatalf("ssz-max not correct, expected 16777216, got %s", sszMax)
	}
}

func TestFullTag(t *testing.T) {
	tag := "`protobuf:\"bytes,1002,opt,name=genesis_validators_root,json=genesisValidatorsRoot,proto3\" json:\"genesis_validators_root,omitempty\" ssz-size:\"32\"`"
	_, err := extractSSZDimensions(tag)
	if err != nil {
		t.Fatalf("unexpected error from extractSSZDimensions: %v", err)
	}
}

func TestListOfVector(t *testing.T) {
	tag := "`protobuf:\"bytes,2004,rep,name=historical_roots,json=historicalRoots,proto3\" json:\"historical_roots,omitempty\" ssz-max:\"16777216\" ssz-size:\"?,32\"`"
	_, err := extractSSZDimensions(tag)
	if err != nil {
		t.Fatalf("unexpected error from extractSSZDimensions: %v", err)
	}
}

func TestWildcardSSZSize(t *testing.T) {
	tag := "`ssz-max:\"16777216\" ssz-size:\"?,32\"`"
	bounds, err := extractSSZDimensions(tag)
	if err != nil {
		t.Fatalf("unexpected error from extractSSZDimensions: %v", err)
	}
	if len(bounds) != 2 {
		t.Fatalf("unexpected number of bounds, want=2, got=%d", len(bounds))
	}
	if !bounds[0].IsList() {
		t.Fatalf("expected first bound to be a list")
	}
	if bounds[0].IsVector() {
		t.Fatalf("expected first bound to not be a vector")
	}
	if bounds[0].ListLen() != 16777216 {
		t.Fatalf("unexpected list length for first bound, want=16777216, got=%d", bounds[0].ListLen())
	}
	if bounds[1].IsList() {
		t.Fatalf("expected second bound to not be a list")
	}
	if !bounds[1].IsVector() {
		t.Fatalf("expected second bound to be a vector")
	}
	if bounds[1].VectorLen() != 32 {
		t.Fatalf("unexpected vector length for second bound, want=32, got=%d", bounds[1].VectorLen())
	}
}

func Test2DWildcardSSZSize(t *testing.T) {
	tag := "`protobuf:\"bytes,14,rep,name=transactions,proto3\" json:\"transactions,omitempty\" ssz-max:\"1048576,1073741824\" ssz-size:\"?,?\"`"
	bounds, err := extractSSZDimensions(tag)
	if err != nil {
		t.Fatalf("unexpected error from extractSSZDimensions: %v", err)
	}
	if len(bounds) != 2 {
		t.Fatalf("unexpected number of bounds, want=2, got=%d", len(bounds))
	}
	if !bounds[0].IsList() {
		t.Fatalf("expected first bound to be a list")
	}
	if bounds[0].IsVector() {
		t.Fatalf("expected first bound to not be a vector")
	}
	if bounds[0].ListLen() != 1048576 {
		t.Fatalf("unexpected list length for first bound, want=1048576, got=%d", bounds[0].ListLen())
	}
	if !bounds[1].IsList() {
		t.Fatalf("expected second bound to be a list")
	}
	if bounds[1].IsVector() {
		t.Fatalf("expected second bound to not be a vector")
	}
	if bounds[1].ListLen() != 1073741824 {
		t.Fatalf("unexpected list length for second bound, want=1073741824, got=%d", bounds[1].ListLen())
	}
}
