package core

import (
	"strings"
	"testing"
)

// The motivating bug: two bare (unaliased) defaults collided on the old flat
// map's "" key, and only one survived into the import block.
func TestImportNamerTwoBareDefaults(t *testing.T) {
	n := NewImportNamer("github.com/example/target", map[string]string{
		"fmt":             "",
		"encoding/binary": "",
	})
	pairs := n.ImportPairs()
	for _, want := range []string{`"fmt"`, `"encoding/binary"`} {
		if !strings.Contains(pairs, want) {
			t.Fatalf("missing %s in import pairs:\n%s", want, pairs)
		}
	}
}

func TestImportNamerCollision(t *testing.T) {
	n := NewImportNamer("github.com/example/target", nil)
	if got := n.NameString("github.com/example/engine/v1"); got != "v1" {
		t.Fatalf("first claimant should get the bare segment, got %q", got)
	}
	if got := n.NameString("github.com/example/widget/v1"); got != "widget_v1" {
		t.Fatalf("collision should extend with the parent segment, got %q", got)
	}
	// stable on repeat lookups
	if got := n.NameString("github.com/example/widget/v1"); got != "widget_v1" {
		t.Fatalf("repeat lookup changed the name: %q", got)
	}
	pairs := n.ImportPairs()
	for _, want := range []string{
		`v1 "github.com/example/engine/v1"`,
		`widget_v1 "github.com/example/widget/v1"`,
	} {
		if !strings.Contains(pairs, want) {
			t.Fatalf("missing %s in import pairs:\n%s", want, pairs)
		}
	}
}

// A bare default claims its package name, so a later package with the same
// base name must disambiguate rather than silently shadow it.
func TestImportNamerBareDefaultClaimsName(t *testing.T) {
	n := NewImportNamer("github.com/example/target", map[string]string{"fmt": ""})
	if got := n.NameString("github.com/example/fmt"); got != "example_fmt" {
		t.Fatalf("expected disambiguated name for second fmt, got %q", got)
	}
}

func TestImportNamerSanitization(t *testing.T) {
	n := NewImportNamer("github.com/example/target", nil)
	if got := n.NameString("github.com/OffchainLabs/go-bitfield"); got != "go_bitfield" {
		t.Fatalf("dash sanitization wrong: %q", got)
	}
	// extreme case: two packages differing only in domain
	if got := n.NameString("github.com/a/pkg"); got != "pkg" {
		t.Fatalf("got %q", got)
	}
	if got := n.NameString("gitlab.com/a/pkg"); got != "a_pkg" {
		t.Fatalf("got %q", got)
	}
	if got := n.NameString("bitbucket.org/a/pkg"); got != "bitbucket_org_a_pkg" {
		t.Fatalf("domain-level disambiguation wrong: %q", got)
	}
}

func TestImportNamerSelf(t *testing.T) {
	n := NewImportNamer("github.com/example/target", nil)
	if got := n.NameString("github.com/example/target"); got != "" {
		t.Fatalf("self reference should be unqualified, got %q", got)
	}
}

func TestImportNamerDeterministic(t *testing.T) {
	build := func() string {
		n := NewImportNamer("github.com/example/target", DefaultSSZImports)
		n.NameString("github.com/example/b")
		n.NameString("github.com/example/a")
		return n.ImportPairs()
	}
	first := build()
	for i := 0; i < 10; i++ {
		if got := build(); got != first {
			t.Fatalf("non-deterministic output:\n%s\nvs\n%s", first, got)
		}
	}
}

// Reserved-but-unused packages must not be emitted: the import block only
// carries packages whose name generation actually requested.
func TestImportNamerReserveUnusedNotEmitted(t *testing.T) {
	n := NewImportNamer("github.com/example/target", map[string]string{"fmt": ""})
	n.Reserve("github.com/example/engine/v1")
	n.Reserve("github.com/example/widget/v1")
	if got := n.NameString("github.com/example/engine/v1"); got != "v1" {
		t.Fatalf("got %q", got)
	}
	pairs := n.ImportPairs()
	if !strings.Contains(pairs, `v1 "github.com/example/engine/v1"`) {
		t.Fatalf("used package missing:\n%s", pairs)
	}
	if strings.Contains(pairs, "widget") {
		t.Fatalf("reserved-but-unused package emitted:\n%s", pairs)
	}
	// seeded defaults are emitted until PruneUnreferenced says otherwise
	if !strings.Contains(pairs, `"fmt"`) {
		t.Fatalf("default missing:\n%s", pairs)
	}
}

func TestImportNamerPruneUnreferenced(t *testing.T) {
	n := NewImportNamer("github.com/example/target", DefaultSSZImports)
	n.NameString("github.com/example/engine/v1")
	n.PruneUnreferenced(`func (c *X) MarshalSSZ() ([]byte, error) {
	if c.Inner == nil {
		c.Inner = new(v1.Inner)
	}
	return nil, fmt.Errorf("nope")
}`)
	pairs := n.ImportPairs()
	for _, want := range []string{`"fmt"`, `v1 "github.com/example/engine/v1"`} {
		if !strings.Contains(pairs, want) {
			t.Fatalf("referenced import %s dropped:\n%s", want, pairs)
		}
	}
	for _, unwanted := range []string{"encoding/binary", "methodical-ssz/ssz"} {
		if strings.Contains(pairs, unwanted) {
			t.Fatalf("unreferenced import %s emitted:\n%s", unwanted, pairs)
		}
	}
}

func TestImportNamerPruneIgnoresCommentsAndStrings(t *testing.T) {
	n := NewImportNamer("github.com/example/target", DefaultSSZImports)
	n.PruneUnreferenced(`// binary.LittleEndian would go here
func (c *X) Err() error { return fmt.Errorf("ssz.ErrBytesLength: %w", nil) }`)
	pairs := n.ImportPairs()
	if !strings.Contains(pairs, `"fmt"`) {
		t.Fatalf("referenced import dropped:\n%s", pairs)
	}
	if strings.Contains(pairs, "encoding/binary") || strings.Contains(pairs, "methodical-ssz/ssz") {
		t.Fatalf("comment/string mention counted as a reference:\n%s", pairs)
	}
}

// Shortest path wins regardless of arrival order: a shorter path arriving
// later steals the bare name from a reserved (not yet used) longer path, which
// is renamed via its ancestor segments.
func TestImportNamerShortestPathWins(t *testing.T) {
	n := NewImportNamer("github.com/example/target", nil)
	n.Reserve("github.com/example/fmt") // longer path arrives first
	n.Reserve("fmt")                    // stdlib arrives second
	if got := n.NameString("fmt"); got != "fmt" {
		t.Fatalf("shortest path should own the bare name, got %q", got)
	}
	if got := n.NameString("github.com/example/fmt"); got != "example_fmt" {
		t.Fatalf("longer path should be renamed, got %q", got)
	}
}

// Once a name has been handed to generated code it is frozen: a later shorter
// arrival must not steal it (the text already references it), and instead
// extends its own candidate — or fails loudly when it cannot.
func TestImportNamerUsedNamesFrozen(t *testing.T) {
	n := NewImportNamer("github.com/example/target", nil)
	if got := n.NameString("github.com/example/sub/thing"); got != "thing" {
		t.Fatalf("got %q", got)
	}
	if got := n.NameString("github.com/other/thing"); got != "other_thing" {
		t.Fatalf("used name must not be stolen, got %q", got)
	}
}

// Steal chains: the renamed victim may itself displace an even deeper
// reserved package.
func TestImportNamerStealChain(t *testing.T) {
	n := NewImportNamer("github.com/example/target", nil)
	n.Reserve("github.com/a/b/example/pkg") // deepest: claims "pkg"
	n.Reserve("github.com/x/example/pkg")   // steals "pkg"; victim renamed "example_pkg"
	n.Reserve("github.com/c/pkg")           // steals "pkg" again
	if got := n.NameString("github.com/c/pkg"); got != "pkg" {
		t.Fatalf("got %q", got)
	}
	if got := n.NameString("github.com/x/example/pkg"); got != "example_pkg" {
		t.Fatalf("got %q", got)
	}
	if got := n.NameString("github.com/a/b/example/pkg"); got != "b_example_pkg" {
		t.Fatalf("got %q", got)
	}
}
