package types

import (
	"go/types"

	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
)

type ValuePointer struct {
	Referent   ValRep
	Interfaces map[*types.Interface]bool
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
	if ti == interfaces.SszUnmarshaler {
		return false
	}
	// since the other methods are read-only, it's ok to use a method with a value receiver
	return vp.Referent.SatisfiesInterface(ti)
}

var _ ValRep = &ValuePointer{}
