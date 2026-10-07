package sszgen

import (
	"strings"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/config"
)

const delegateFixturePkg = "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture"

func scoperForConfig(t *testing.T, gc *config.GeneratorConfig) PathScoper {
	t.Helper()
	gc.TypeMap = make(map[string]config.TypeConfig, len(gc.Types))
	for _, ty := range gc.Types {
		gc.TypeMap[ty.Name] = ty
	}
	ps, err := NewGoPathScoper(gc.Package, gc)
	if err != nil {
		t.Fatal(err)
	}
	return ps
}

// A misspelled field key must fail loudly rather than be silently ignored.
func TestValidateConfigUnknownField(t *testing.T) {
	gc := &config.GeneratorConfig{
		Package: delegateFixturePkg,
		Types: []config.TypeConfig{{
			Name: "DelegateContainer",
			Fields: map[string]config.FieldConfig{
				"Blbos": {Type: config.FieldTypeProgressiveList}, // typo of Blobs
			},
		}},
	}
	ps := scoperForConfig(t, gc)
	_, err := TypeDefs(ps, "DelegateContainer")
	if err == nil {
		t.Fatal("expected an error for an unknown field name in config")
	}
	if !strings.Contains(err.Error(), "Blbos") {
		t.Fatalf("error should name the offending field, got: %v", err)
	}
}

// A correct config must validate cleanly (and not change existing behavior).
func TestValidateConfigKnownField(t *testing.T) {
	gc := &config.GeneratorConfig{
		Package: delegateFixturePkg,
		Types: []config.TypeConfig{{
			Name: "DelegateContainer",
			Fields: map[string]config.FieldConfig{
				"Blobs": {Type: config.FieldTypeProgressiveList},
			},
		}},
	}
	ps := scoperForConfig(t, gc)
	if _, err := TypeDefs(ps, "DelegateContainer"); err != nil {
		t.Fatalf("valid config should not error, got: %v", err)
	}
}

// Field/progressive config on a non-struct type cannot take effect and must error.
func TestValidateConfigOnNonStruct(t *testing.T) {
	gc := &config.GeneratorConfig{
		Package: delegateFixturePkg,
		Types: []config.TypeConfig{{
			Name:        "Light", // a `uint64` named type, not a struct
			Progressive: &config.ProgressiveConfig{},
		}},
	}
	ps := scoperForConfig(t, gc)
	_, err := TypeDefs(ps, "Light")
	if err == nil {
		t.Fatal("expected an error for progressive config on a non-struct type")
	}
	if !strings.Contains(err.Error(), "Light") {
		t.Fatalf("error should name the offending type, got: %v", err)
	}
}

// Multiple typos across types are aggregated into a single error.
func TestValidateConfigAggregatesErrors(t *testing.T) {
	gc := &config.GeneratorConfig{
		Package: delegateFixturePkg,
		Types: []config.TypeConfig{{
			Name: "DelegateContainer",
			Fields: map[string]config.FieldConfig{
				"Nope":  {Type: config.FieldTypeProgressiveList},
				"Maybe": {Type: config.FieldTypeProgressiveList},
			},
		}},
	}
	ps := scoperForConfig(t, gc)
	_, err := TypeDefs(ps, "DelegateContainer")
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"Nope", "Maybe"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("aggregated error should mention %q, got: %v", want, err)
		}
	}
}
