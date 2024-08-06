package types

import "go/types"

type ValueList struct {
	ElementValue ValRep
	MaxSize      int
	// Progressive marks an SSZ ProgressiveList (or the underlying list of a
	// ProgressiveBitlist): merkleized progressively, with no limit — MaxSize
	// is unused when set.
	Progressive bool
}

func (vl *ValueList) TypeName() string {
	return "[]" + vl.ElementValue.TypeName()
}

func (vl *ValueList) PackagePath() string {
	return vl.ElementValue.PackagePath()
}

func (vl *ValueList) FixedSize() int {
	return 4
}

func (vl *ValueList) IsVariableSized() bool {
	return true
}

func (vl *ValueList) SatisfiesInterface(*types.Interface) bool {
	return false
}

var _ ValRep = &ValueList{}
