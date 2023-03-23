package types

import "go/types"

type ValRep interface {
	TypeName() string
	FixedSize() int
	PackagePath() string
	IsVariableSized() bool
	SatisfiesInterface(*types.Interface) bool
}
