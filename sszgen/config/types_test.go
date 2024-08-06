package config

import "testing"

func TestProgressiveActiveFields(t *testing.T) {
	cases := []struct {
		name      string
		inactive  []int
		numFields int
		want      []bool
		wantErr   bool
	}{
		{name: "empty config all active", numFields: 3, want: []bool{true, true, true}},
		{name: "gap", inactive: []int{1}, numFields: 2, want: []bool{true, false, true}},
		{name: "multiple gaps", inactive: []int{1, 3}, numFields: 3, want: []bool{true, false, true, false, true}},
		{name: "no fields", numFields: 0, wantErr: true},
		{name: "trailing zero", inactive: []int{2}, numFields: 2, wantErr: true},
		{name: "out of range", inactive: []int{7}, numFields: 2, wantErr: true},
		{name: "negative", inactive: []int{-1}, numFields: 2, wantErr: true},
		{name: "duplicate", inactive: []int{1, 1}, numFields: 3, wantErr: true},
		{name: "over 256 bits", numFields: 257, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &ProgressiveConfig{InactiveIndices: c.inactive}
			got, err := p.ActiveFields(c.numFields)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("want length %d, got %d", len(c.want), len(got))
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("position %d: want %v got %v (full: %v)", i, c.want[i], got[i], got)
				}
			}
		})
	}
}

func TestParseGeneratorConfigProgressive(t *testing.T) {
	yml := `
package: github.com/example/foo
types:
  - name: Foo
    progressive: {}
    fields:
      Balances: {type: "ProgressiveList"}
      Bits:     {type: "ProgressiveBitlist"}
  - name: Bar
    progressive:
      inactive_indices: [1, 5]
`
	g, err := ParseGeneratorConfig([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	foo := g.TypeConfig("Foo")
	if foo.Progressive == nil {
		t.Fatal("Foo should be progressive")
	}
	if len(foo.Progressive.InactiveIndices) != 0 {
		t.Fatalf("Foo should have no inactive indices, got %v", foo.Progressive.InactiveIndices)
	}
	if foo.Fields["Balances"].Type != FieldTypeProgressiveList {
		t.Fatalf("unexpected field type %q", foo.Fields["Balances"].Type)
	}
	if foo.Fields["Bits"].Type != FieldTypeProgressiveBitlist {
		t.Fatalf("unexpected field type %q", foo.Fields["Bits"].Type)
	}
	bar := g.TypeConfig("Bar")
	if bar.Progressive == nil || len(bar.Progressive.InactiveIndices) != 2 {
		t.Fatalf("Bar progressive config wrong: %+v", bar.Progressive)
	}
	if g.TypeConfig("Baz").Progressive != nil {
		t.Fatal("unknown type should not be progressive")
	}
}

func TestDisableProgressive(t *testing.T) {
	yml := `
package: github.com/example/foo
types:
  - name: Foo
    progressive: {}
    fields:
      Balances: {type: "ProgressiveList"}
      Bits:     {type: "ProgressiveBitlist"}
      Txs:      {type: "ProgressiveList", element: {type: "ProgressiveByteList"}}
  - name: Bar
    progressive:
      inactive_indices: [1, 5]
`
	g, err := ParseGeneratorConfig([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	g.DisableProgressive()

	// Every progressive-container marker must be cleared.
	if foo := g.TypeConfig("Foo"); foo.Progressive != nil {
		t.Fatalf("Foo.Progressive should be nil after DisableProgressive, got %+v", foo.Progressive)
	}
	if bar := g.TypeConfig("Bar"); bar.Progressive != nil {
		t.Fatalf("Bar.Progressive should be nil after DisableProgressive, got %+v", bar.Progressive)
	}

	// Every progressive field override (including nested element configs) must
	// be dropped so the field falls back to its standard bounded form.
	if fields := g.TypeConfig("Foo").Fields; len(fields) != 0 {
		t.Fatalf("Foo.Fields should be empty after DisableProgressive, got %+v", fields)
	}

	// The mutation must be visible through both the slice and the TypeMap.
	for i := range g.Types {
		if g.Types[i].Progressive != nil {
			t.Fatalf("Types[%d].Progressive should be nil, got %+v", i, g.Types[i].Progressive)
		}
		for name, fc := range g.Types[i].Fields {
			if isProgressiveFieldType(fc.Type) {
				t.Fatalf("Types[%d] still has progressive field %q: %+v", i, name, fc)
			}
		}
	}
}
