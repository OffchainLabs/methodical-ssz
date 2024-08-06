package sszgen

import (
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/config"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// The yaml config drives the progressive frontend: a `progressive` object
// marks the container (computing its active-fields bitvector), and `fields`
// entries override individual fields as progressive collections (which need
// no ssz-max tag). Reuses the compiled delegatefixture package through the
// production GoPathScoper.
func TestProgressiveConfigFrontend(t *testing.T) {
	pkgPath := "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture"
	gc := &config.GeneratorConfig{
		Package: pkgPath,
		Types: []config.TypeConfig{{
			Name:        "DelegateContainer",
			Progressive: &config.ProgressiveConfig{InactiveIndices: []int{2}},
			Fields: map[string]config.FieldConfig{
				"Blobs": {Type: config.FieldTypeProgressiveList},
			},
		}},
	}
	gc.TypeMap = map[string]config.TypeConfig{gc.Types[0].Name: gc.Types[0]}

	ps, err := NewGoPathScoper(pkgPath, gc)
	if err != nil {
		t.Fatal(err)
	}
	defs, err := TypeDefs(ps, "DelegateContainer")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatal(err)
	}
	vc, ok := rep.(*gentypes.ValueContainer)
	if !ok {
		t.Fatalf("want *ValueContainer, got %T", rep)
	}
	// 5 Go fields + 1 inactive index = 6 positions, index 2 inactive
	want := []bool{true, true, false, true, true, true}
	if len(vc.ActiveFields) != len(want) {
		t.Fatalf("want %d active-field positions, got %d", len(want), len(vc.ActiveFields))
	}
	for i := range want {
		if vc.ActiveFields[i] != want[i] {
			t.Fatalf("active fields position %d: want %v, got %v", i, want[i], vc.ActiveFields[i])
		}
	}
	blobs, err := vc.GetField("Blobs")
	if err != nil {
		t.Fatal(err)
	}
	list, ok := blobs.(*gentypes.ValueList)
	if !ok {
		t.Fatalf("Blobs: want *ValueList, got %T", blobs)
	}
	if !list.Progressive {
		t.Fatal("Blobs should be a progressive list")
	}
	if _, ok := list.ElementValue.(*gentypes.ValuePointer); !ok {
		t.Fatalf("Blobs element: want *ValuePointer, got %T", list.ElementValue)
	}
}

// TestDisableProgressiveDowngrade exercises the full --disable-progressive path
// through the real parser: after GeneratorConfig.DisableProgressive the same
// config that produces a progressive container with a progressive Blobs list
// instead yields a standard container (no active-fields) whose Blobs field is a
// bounded list, with MaxSize recovered from the field's ssz-max:"256" tag.
func TestDisableProgressiveDowngrade(t *testing.T) {
	pkgPath := "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture"
	newCfg := func() *config.GeneratorConfig {
		gc := &config.GeneratorConfig{
			Package: pkgPath,
			Types: []config.TypeConfig{{
				Name:        "DelegateContainer",
				Progressive: &config.ProgressiveConfig{InactiveIndices: []int{2}},
				Fields: map[string]config.FieldConfig{
					"Blobs": {Type: config.FieldTypeProgressiveList},
				},
			}},
		}
		gc.TypeMap = map[string]config.TypeConfig{gc.Types[0].Name: gc.Types[0]}
		return gc
	}
	parse := func(gc *config.GeneratorConfig) *gentypes.ValueContainer {
		t.Helper()
		ps, err := NewGoPathScoper(pkgPath, gc)
		if err != nil {
			t.Fatal(err)
		}
		defs, err := TypeDefs(ps, "DelegateContainer")
		if err != nil {
			t.Fatal(err)
		}
		rep, err := ParseTypeDef(defs[0])
		if err != nil {
			t.Fatal(err)
		}
		vc, ok := rep.(*gentypes.ValueContainer)
		if !ok {
			t.Fatalf("want *ValueContainer, got %T", rep)
		}
		return vc
	}
	blobsList := func(vc *gentypes.ValueContainer) *gentypes.ValueList {
		t.Helper()
		blobs, err := vc.GetField("Blobs")
		if err != nil {
			t.Fatal(err)
		}
		list, ok := blobs.(*gentypes.ValueList)
		if !ok {
			t.Fatalf("Blobs: want *ValueList, got %T", blobs)
		}
		return list
	}

	// Baseline: the progressive config yields a progressive container + list.
	base := parse(newCfg())
	if len(base.ActiveFields) == 0 {
		t.Fatal("baseline container should carry active fields")
	}
	if !blobsList(base).Progressive {
		t.Fatal("baseline Blobs should be a progressive list")
	}

	// Downgraded: no active fields, and Blobs is a standard bounded list.
	gc := newCfg()
	gc.DisableProgressive()
	down := parse(gc)
	if len(down.ActiveFields) != 0 {
		t.Fatalf("downgraded container should have no active fields, got %v", down.ActiveFields)
	}
	dl := blobsList(down)
	if dl.Progressive {
		t.Fatal("downgraded Blobs should not be progressive")
	}
	if dl.MaxSize != 256 {
		t.Fatalf("downgraded Blobs should be bounded with MaxSize 256, got %d", dl.MaxSize)
	}
}

// applyElementConfig drives nested progressive collections that have no named
// Go type to delegate to (e.g. a ProgressiveList of ProgressiveByteLists, where
// the element is an inline [][]byte inner slice). The bounded expansion resolves
// the element's dimensions; applyElementConfig only flips the resolved element
// to its progressive form.
func TestApplyElementConfig(t *testing.T) {
	byteEl := func() gentypes.ValRep { return &gentypes.ValueByte{Name: "byte"} }
	// element resolved by the bounded expansion: a ByteList (list of byte, bounded).
	boundedByteList := func() *gentypes.ValueList {
		return &gentypes.ValueList{ElementValue: byteEl(), MaxSize: 256}
	}

	t.Run("nil leaves element unchanged", func(t *testing.T) {
		in := boundedByteList()
		got, err := applyElementConfig("f", in, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got != gentypes.ValRep(in) {
			t.Fatalf("want element returned unchanged")
		}
	})

	t.Run("ProgressiveByteList element flips to progressive, drops MaxSize", func(t *testing.T) {
		got, err := applyElementConfig("Transactions", boundedByteList(),
			&config.FieldConfig{Type: config.FieldTypeProgressiveByteList})
		if err != nil {
			t.Fatal(err)
		}
		gl, ok := got.(*gentypes.ValueList)
		if !ok {
			t.Fatalf("want *ValueList, got %T", got)
		}
		if !gl.Progressive {
			t.Fatal("element list should be progressive")
		}
		if gl.MaxSize != 0 {
			t.Fatalf("progressive list must not carry MaxSize, got %d", gl.MaxSize)
		}
		if _, ok := gl.ElementValue.(*gentypes.ValueByte); !ok {
			t.Fatalf("want byte element, got %T", gl.ElementValue)
		}
	})

	t.Run("ProgressiveByteList on non-byte element errors", func(t *testing.T) {
		nonByte := &gentypes.ValueList{ElementValue: &gentypes.ValueUint{Size: 64}, MaxSize: 256}
		if _, err := applyElementConfig("f", nonByte,
			&config.FieldConfig{Type: config.FieldTypeProgressiveByteList}); err == nil {
			t.Fatal("want error for ProgressiveByteList over non-byte element")
		}
	})

	t.Run("element config on non-list element errors", func(t *testing.T) {
		if _, err := applyElementConfig("f", byteEl(),
			&config.FieldConfig{Type: config.FieldTypeProgressiveList}); err == nil {
			t.Fatal("want error for list element config over non-list")
		}
	})
}
