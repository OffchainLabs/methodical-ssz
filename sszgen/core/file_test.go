package core

import "testing"

func TestRenderedPackageName(t *testing.T) {
	before := "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	after := "v1"
	got := RenderedPackageName(before)
	if got != after {
		t.Fatalf("unexpected result for RenderedPackageName, want=%s, got=%s", after, got)
	}
}
