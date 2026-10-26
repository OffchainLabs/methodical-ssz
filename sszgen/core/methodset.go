package core

import gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"

// MethodSet is one group of generated methods for a container type (e.g. all the
// size methods, or all the HTR methods). Generate returns the code blocks for vc.
type MethodSet struct {
	Name     string
	Generate func(vc *gentypes.ValueContainer, ctx *GenContext) ([]string, error)
}
