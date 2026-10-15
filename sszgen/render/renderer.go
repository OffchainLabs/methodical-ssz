// Package render hosts the SSZ code generator: one file per operation (size,
// marshal, unmarshal, htr), each implementing the core.Op visitor over every
// SSZ kind, plus the Render orchestration that assembles a complete file from
// an ordered list of method sets (see PLUGIN-SEAM.md / DESIGN.md).
package render

import (
	"fmt"

	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// DefaultMethodSets returns the SSZ method sets in canonical file order, which
// is part of the generated file's shape.
func DefaultMethodSets() []core.MethodSet {
	return []core.MethodSet{
		SizeMethodSet(),
		MarshalMethodSet(),
		UnmarshalMethodSet(),
		HashTreeRootMethodSet(),
		ProgressiveHashTreeRootMethodSet(),
	}
}

// Render generates the complete SSZ source file for vrs by running every method
// set (DefaultMethodSets) over each type in order, sharing one ImportNamer, then
// assembling one file. This is the single entry point used by the CLI and tests,
// so the tested path is the shipped path.
//
// packageNameOverride, when empty, defaults to the last element of packagePath.
func Render(packagePath, packageNameOverride string, vrs []gentypes.ValRep) ([]byte, error) {
	if packagePath == "" && packageNameOverride == "" {
		return nil, fmt.Errorf("missing packagePath: Render requires a packagePath for code generation")
	}
	namer := core.NewImportNamer(packagePath, core.DefaultSSZImports)
	// the default interface set is a process-wide singleton, so its identity
	// keys match the support maps the frontend built (see interfaces.Set)
	ifaces, err := interfaces.DefaultSet()
	if err != nil {
		return nil, err
	}
	ctx := &core.GenContext{
		TargetPackage: packagePath,
		Namer:         namer,
		Interfaces:    ifaces,
	}
	methodSets := DefaultMethodSets()

	// reserve every reachable package before any text is rendered, so alias
	// assignment (shortest-path-wins) is independent of generation order
	core.ReserveImports(namer, vrs)

	blocks := make([]string, 0)
	for _, vr := range vrs {
		vc, ok := vr.(*gentypes.ValueContainer)
		if !ok {
			// Top-level overlays generate no methods today (legacy parity); other
			// kinds are unsupported as top-level generation targets.
			if _, isOverlay := vr.(*gentypes.ValueOverlay); isOverlay {
				continue
			}
			return nil, fmt.Errorf("can only generate methods for container & overlay types, got %T", vr)
		}
		for _, ms := range methodSets {
			b, err := ms.Generate(vc, ctx)
			if err != nil {
				return nil, fmt.Errorf("method set %q: %w", ms.Name, err)
			}
			blocks = append(blocks, b...)
		}
	}

	pkgName := packageNameOverride
	if pkgName == "" {
		pkgName = core.RenderedPackageName(packagePath)
	}
	return core.RenderFile(pkgName, namer, blocks)
}
