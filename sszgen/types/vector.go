package types

import "go/types"

type ValueVector struct {
	ElementValue ValRep
	Size         int
	IsArray      bool
}

func (vv *ValueVector) TypeName() string {
	return "[]" + vv.ElementValue.TypeName()
}

func (vv *ValueVector) FixedSize() int {
	if vv.IsVariableSized() {
		return 4
	}
	return vv.Size * vv.ElementValue.FixedSize()
}

func (vv *ValueVector) PackagePath() string {
	return vv.ElementValue.PackagePath()
}

func (vv *ValueVector) IsVariableSized() bool {
	return vv.ElementValue.IsVariableSized()
}

func (vv *ValueVector) SatisfiesInterface(*types.Interface) bool {
	return false
}

var _ ValRep = &ValueVector{}
