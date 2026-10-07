package sszgen

import (
	"flag"
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/render"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

var update = flag.Bool("update", false, "update golden files")

// TestDelegateFixture renders the delegatefixture package — real, compiled
// types implementing the SSZ runtime interfaces — through the production
// GoPathScoper, exercising the delegation paths no other fixture reaches: the
// full-hasher branch (HashTreeRootWith), value-receiver delegation, and
// delegated elements (fixed and variable) inside collections.
func TestDelegateFixture(t *testing.T) {
	pkgPath := "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture"
	ps, err := NewGoPathScoper(pkgPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defs, err := TypeDefs(ps, "DelegateContainer")
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 type def, got %d", len(defs))
	}
	rep, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatal(err)
	}
	got, err := render.Render(pkgPath, "", []gentypes.ValRep{rep})
	if err != nil {
		t.Fatal(err)
	}

	golden := "testdata/delegate.ssz.go.expected"
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("reading golden (run with -update to create it): %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("Render output differs from golden %s (run with -update if the change is intended):\n--- want ---\n%s\n--- got ---\n%s", golden, want, got)
	}
}
