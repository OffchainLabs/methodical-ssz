package types

import "go/types"

type ValueUnion struct {
	Name string
}

func (vu *ValueUnion) TypeName() string {
	return vu.Name
}

func (vu *ValueUnion) PackagePath() string {
	panic("not implemented")
}

func (vu *ValueUnion) FixedSize() int {
	panic("not implemented")
}

func (vu *ValueUnion) IsVariableSized() bool {
	panic("not implemented")
}

func (vu *ValueUnion) SatisfiesInterface(*types.Interface) bool {
	return false
}

var _ ValRep = &ValueUnion{}
