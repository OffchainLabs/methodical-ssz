package backend

import (
	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

type generateUnion struct {
	*types.ValueUnion
	targetPackage string
	importNamer   *ImportNamer
}

func (g *generateUnion) generateHTRPutter(fieldName string) string {
	return ""
}

func (g *generateUnion) generateUnmarshalValue(fieldName string, s string) string {
	return ""
}

func (g *generateUnion) generateFixedMarshalValue(fieldName string) string {
	return ""
}

func (g *generateUnion) variableSizeSSZ(fieldname string) string {
	return ""
}

var _ valueGenerator = &generateUnion{}
