package core

import gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"

// Coerce returns a Rewriter that casts a field-reference expression to vr's Go
// type — e.g. uint64(x) or []byte(x) — for the SSZ kinds whose named/overlay
// wrapper must be unwrapped before a primitive hasher/marshal call. For other
// kinds it returns the identity.
//
// It is applied to the ref a parent op hands a child via Dispatch, to unwrap
// overlay fields before a primitive call.
func Coerce(vr gentypes.ValRep) Rewriter {
	// A named array needs no conversion: indexing, slicing, and ranging apply
	// to it directly, and a conversion would actually break the array paths —
	// the result of a conversion is unaddressable, so `[32]byte(x)[:]` does
	// not compile.
	if v, ok := vr.(*gentypes.ValueVector); ok && v.IsArray {
		return func(s string) string { return s }
	}
	switch vr.(type) {
	case *gentypes.ValueUint, *gentypes.ValueByte, *gentypes.ValueBool, *gentypes.ValueVector:
		tn := vr.TypeName()
		return func(s string) string { return tn + "(" + s + ")" }
	default:
		return func(s string) string { return s }
	}
}
