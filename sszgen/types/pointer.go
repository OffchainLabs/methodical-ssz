package types

import (
	"go/types"
)

type ValuePointer struct {
	Referent   ValRep
	Interfaces map[*types.Interface]bool
	// Unmarshaler is the unmarshal-delegation interface of the Set used to
	// build Interfaces (an identity key). SatisfiesInterface needs to know
	// which queried interface is the unmarshaler so it can refuse to fall
	// through to the referent for it — UnmarshalSSZ requires a pointer
	// receiver, so a miss in the pointer's own method-set map is final.
	Unmarshaler *types.Interface
}

func (vp *ValuePointer) TypeName() string {
	return "*" + vp.Referent.TypeName()
}

func (vp *ValuePointer) PackagePath() string {
	return vp.Referent.PackagePath()
}

func (vp *ValuePointer) FixedSize() int {
	return vp.Referent.FixedSize()
}

func (vp *ValuePointer) IsVariableSized() bool {
	return vp.Referent.IsVariableSized()
}

func (vp *ValuePointer) SatisfiesInterface(ti *types.Interface) bool {
	if vp.Interfaces != nil && vp.Interfaces[ti] {
		return true
	}
	// Unmarshaler needs a pointer receiver, and the above check failed means that there isn't one,
	// so we shouldn't allow a value receiver to satisfy the interface.
	if ti == vp.Unmarshaler {
		return false
	}
	// since the other methods are read-only, it's ok to use a method with a value receiver
	return vp.Referent.SatisfiesInterface(ti)
}

var _ ValRep = &ValuePointer{}
