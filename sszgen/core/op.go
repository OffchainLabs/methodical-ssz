package core

import (
	"fmt"
	"go/types"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// Rewriter transforms a field-reference expression — e.g. wrapping it in a cast,
// uint64(x). It is the inline seam used by system generators like coerce, applied
// to the ref a parent op hands to a child via Dispatch.
type Rewriter = func(string) string

// Op is one codegen operation (size, htr, ...) expressed as a visitor over the
// SSZ value kinds: one method per kind, plus Delegate for field types that already
// implement the operation's runtime interface. Each method returns the operation's
// fragment F for a single field, given the threaded reference R.
//
// R is what the op carries down to a child via Dispatch. For most operations it is
// just the field-reference expression (a string); for unmarshal it bundles the
// destination field, the source byte slice, and an output caster. F is the
// per-kind output shape.
//
// The shared Dispatch holds the routing + delegation logic, so an op writes only
// the per-kind bodies — the whole operation lives in one place.
type Op[F, R any] interface {
	Name() string
	// Delegates is the runtime interface that, when a field type already
	// implements it, routes the field to Delegate instead of an inlined per-kind
	// body. The interface is resolved from the run's GenContext (ctx.Ifaces()),
	// so the identity keys match the support maps the frontend built — and so a
	// future plugin op can target its own interface set. Return nil to disable
	// delegation.
	Delegates(ctx *GenContext) *types.Interface
	Delegate(vr gentypes.ValRep, ref R, ctx *GenContext) F

	Bool(v *gentypes.ValueBool, ref R, ctx *GenContext) F
	Byte(v *gentypes.ValueByte, ref R, ctx *GenContext) F
	Uint(v *gentypes.ValueUint, ref R, ctx *GenContext) F
	Vector(v *gentypes.ValueVector, ref R, ctx *GenContext) F
	List(v *gentypes.ValueList, ref R, ctx *GenContext) F
	Container(v *gentypes.ValueContainer, ref R, ctx *GenContext) F
	Overlay(v *gentypes.ValueOverlay, ref R, ctx *GenContext) F
	Pointer(v *gentypes.ValuePointer, ref R, ctx *GenContext) F
	Union(v *gentypes.ValueUnion, ref R, ctx *GenContext) F
}

// Dispatch routes vr to op's matching per-kind method — or to op.Delegate when vr
// already implements op.Delegates(ctx) — passing the threaded reference ref. It is
// the single recursion primitive an op uses to descend into element, field,
// referent, and underlying types.
func Dispatch[F, R any](op Op[F, R], vr gentypes.ValRep, ref R, ctx *GenContext) F {
	if iface := op.Delegates(ctx); iface != nil && vr.SatisfiesInterface(iface) {
		return op.Delegate(vr, ref, ctx)
	}
	switch v := vr.(type) {
	case *gentypes.ValueBool:
		return op.Bool(v, ref, ctx)
	case *gentypes.ValueByte:
		return op.Byte(v, ref, ctx)
	case *gentypes.ValueUint:
		return op.Uint(v, ref, ctx)
	case *gentypes.ValueVector:
		return op.Vector(v, ref, ctx)
	case *gentypes.ValueList:
		return op.List(v, ref, ctx)
	case *gentypes.ValueContainer:
		return op.Container(v, ref, ctx)
	case *gentypes.ValueOverlay:
		return op.Overlay(v, ref, ctx)
	case *gentypes.ValuePointer:
		return op.Pointer(v, ref, ctx)
	case *gentypes.ValueUnion:
		return op.Union(v, ref, ctx)
	default:
		panic(fmt.Sprintf("core.Dispatch: unsupported ValRep %T", vr))
	}
}
