package types

import (
	"fmt"
	"go/types"
)

type ValueOverlay struct {
	Name       string
	Package    string
	Underlying ValRep
	Interfaces map[*types.Interface]bool
}

func (vo *ValueOverlay) TypeName() string {
	return vo.Name
}

func (vo *ValueOverlay) PackagePath() string {
	return vo.Package
}

func (vo *ValueOverlay) FixedSize() int {
	if vo.IsBitfield() {
		return vo.bitfieldFixedSize()
	}
	return vo.Underlying.FixedSize()
}

func (vo *ValueOverlay) IsVariableSized() bool {
	return vo.Underlying.IsVariableSized()
}

func (vo *ValueOverlay) SatisfiesInterface(ti *types.Interface) bool {
	return vo.Interfaces != nil && vo.Interfaces[ti]
}

func (vo *ValueOverlay) IsBitfield() bool {
	return vo.Package == "github.com/prysmaticlabs/go-bitfield"
}

func (vo *ValueOverlay) bitfieldFixedSize() int {
	switch vo.Name {
	case "Bitlist":
		return 4
	case "Bitlist64":
		return 4
	case "Bitvector4":
		return 1
	case "Bitvector8":
		return 1
	case "Bitvector32":
		return 4
	case "Bitvector64":
		return 8
	case "Bitvector128":
		return 16
	case "Bitvector256":
		return 32
	case "Bitvector512":
		return 64
	case "Bitvector1024":
		return 128
	}
	panic(fmt.Sprintf("Can't determine the correct size for bitfield type = %s", vo.Name))
}

var _ ValRep = &ValueOverlay{}
