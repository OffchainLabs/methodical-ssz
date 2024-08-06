package sszgen

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// validateConfigs cross-checks the generator config against the parsed types,
// failing on config that can never take effect (usually a mistyped field name,
// which is otherwise silently ignored). Errors across all types are aggregated.
func validateConfigs(defs []*TypeDef) error {
	errs := make([]error, 0, len(defs))
	for _, def := range defs {
		if err := validateTypeConfig(def); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// validateTypeConfig verifies that every override in a single type's config
// refers to something real. For a struct, each key in cfg.Fields must name an
// exported field. A non-struct type cannot carry field or progressive-container
// config at all, so any is reported as a mistake.
func validateTypeConfig(def *TypeDef) error {
	if !def.IsStruct {
		var bad []string
		if len(def.cfg.Fields) > 0 {
			bad = append(bad, "fields")
		}
		if def.cfg.Progressive != nil {
			bad = append(bad, "progressive")
		}
		if len(bad) == 0 {
			return nil
		}
		return fmt.Errorf("type %s: %s config is only valid on struct types", def.Name, strings.Join(bad, " and "))
	}

	if len(def.cfg.Fields) == 0 {
		return nil
	}
	known := make(map[string]bool, len(def.Fields))
	for _, f := range def.Fields {
		known[f.name] = true
	}
	var unknown []string
	for name := range def.cfg.Fields {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("type %s: field config references unknown field(s) %v; valid fields are %v",
		def.Name, unknown, sortedFieldNames(def.Fields))
}

func sortedFieldNames(fields []*FieldDef) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.name
	}
	sort.Strings(names)
	return names
}
