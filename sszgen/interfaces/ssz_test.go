package interfaces

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestLoadPackage(t *testing.T) {
	// Load the package.
	pkg, err := loadSsz()
	if err != nil {
		t.Fatal(err)
	}

	// Check the package.
	if pkg == nil {
		t.Fatal("expected a package")
	}
}

// DefaultSet must return the same instance on every call: its pointers are
// identity keys shared between the frontend's support maps and the render
// GenContext, so a per-call instance would silently disable delegation.
func TestDefaultSetSingleton(t *testing.T) {
	a, err := DefaultSet()
	if err != nil {
		t.Fatal(err)
	}
	b, err := DefaultSet()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("DefaultSet returned distinct instances")
	}
	for name, iface := range map[string]interface{ NumMethods() int }{
		"Marshaler":   a.Marshaler,
		"Unmarshaler": a.Unmarshaler,
		"FullHasher":  a.FullHasher,
		"LightHasher": a.LightHasher,
		"Sizer":       a.Sizer,
	} {
		if iface == nil || iface.NumMethods() == 0 {
			t.Fatalf("interface %s missing or empty", name)
		}
	}
}

// A Set built from a separate packages.Load is a separate key universe: none
// of its pointers are identical to the default set's, which is exactly why one
// run must use a single Set throughout.
func TestNewSetDistinctIdentity(t *testing.T) {
	pkg, err := loadSsz()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSet(pkg.Types)
	if err != nil {
		t.Fatal(err)
	}
	d, err := DefaultSet()
	if err != nil {
		t.Fatal(err)
	}
	if s.Marshaler == d.Marshaler || s.Unmarshaler == d.Unmarshaler ||
		s.FullHasher == d.FullHasher || s.LightHasher == d.LightHasher || s.Sizer == d.Sizer {
		t.Fatal("expected interfaces from a separate load to be distinct from the default set's")
	}
}

// SupportMap is the foundation of delegation: it decides, per interface, which
// generated code delegates instead of inlining. Check it directly against the
// in-module fixture types, whose method sets are purpose-built: Wide implements
// everything including the full hasher (pointer receivers); Light implements
// the marshal set and light hasher with value receivers (Unmarshal necessarily
// pointer); Blob implements everything but the full hasher. The Set is built
// from the ssz package inside the SAME load universe as the fixture types, the
// way a production run sees them.
func TestSupportMap(t *testing.T) {
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports}
	pkgs, err := packages.Load(cfg, "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("want 1 package, got %d", len(pkgs))
	}
	fixture := pkgs[0].Types

	sszPkg := findImport(fixture, sszPkgPath, make(map[*types.Package]bool))
	if sszPkg == nil {
		t.Fatal("ssz package not found in fixture's load universe")
	}
	set, err := NewSet(sszPkg)
	if err != nil {
		t.Fatal(err)
	}

	lookup := func(name string) types.Type {
		t.Helper()
		obj := fixture.Scope().Lookup(name)
		if obj == nil {
			t.Fatalf("type %s not found in fixture package", name)
		}
		return obj.Type()
	}

	type expectation struct {
		marshaler, unmarshaler, sizer, light, full bool
	}
	cases := []struct {
		name string
		typ  types.Type
		want expectation
	}{
		// pointer receivers; full SSZ method set including HashTreeRootWith
		{"Wide", lookup("Wide"), expectation{marshaler: true, unmarshaler: true, sizer: true, light: true, full: true}},
		// value receivers (Unmarshal pointer, satisfied via the NewPointer fallback)
		{"Light", lookup("Light"), expectation{marshaler: true, unmarshaler: true, sizer: true, light: true, full: false}},
		// pointer receivers, light hasher only
		{"Blob", lookup("Blob"), expectation{marshaler: true, unmarshaler: true, sizer: true, light: true, full: false}},
		// no methods at all
		{"no methods", types.NewNamed(types.NewTypeName(0, fixture, "Bare", nil), types.Typ[types.Uint64], nil), expectation{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := set.SupportMap(c.typ)
			got := expectation{
				marshaler:   m[set.Marshaler],
				unmarshaler: m[set.Unmarshaler],
				sizer:       m[set.Sizer],
				light:       m[set.LightHasher],
				full:        m[set.FullHasher],
			}
			if got != c.want {
				t.Fatalf("support map mismatch:\nwant %+v\ngot  %+v", c.want, got)
			}
		})
	}
}
