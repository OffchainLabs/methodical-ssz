package core

import (
	"sort"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// ReserveImports walks the codegen target types once and reserves every
// reachable package with the namer, before any method set renders text. With
// all packages known up front, the namer's shortest-path-wins naming can
// reorganize aliases freely (nothing is templated yet), so name assignment is
// independent of the order generation happens to touch packages. The walk
// over-approximates — not every reachable package is referenced by name in
// generated code — which is harmless: only packages whose name generation
// actually requests (NameString) are emitted in the import block.
func ReserveImports(n *ImportNamer, vrs []gentypes.ValRep) {
	paths := make(map[string]bool)
	var visit func(vr gentypes.ValRep)
	visit = func(vr gentypes.ValRep) {
		if p := vr.PackagePath(); p != "" {
			paths[p] = true
		}
		switch v := vr.(type) {
		case *gentypes.ValueContainer:
			for _, f := range v.Contents {
				visit(f.Value)
			}
		case *gentypes.ValueList:
			visit(v.ElementValue)
		case *gentypes.ValueVector:
			visit(v.ElementValue)
		case *gentypes.ValuePointer:
			visit(v.Referent)
		case *gentypes.ValueOverlay:
			visit(v.Underlying)
		}
	}
	for _, vr := range vrs {
		visit(vr)
	}

	// sorted for deterministic reservation order (the steal policy makes the
	// final names order-independent for differing depths; sorting pins ties)
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)
	for _, p := range sorted {
		n.Reserve(p)
	}
}
