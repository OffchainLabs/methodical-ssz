package backend

import (
	"fmt"

	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

type generatePointer struct {
	*types.ValuePointer
	targetPackage string
	importNamer   *ImportNamer
}

func (g *generatePointer) generateHTRPutter(fieldName string) string {
	gg := newValueGenerator(interfaces.SszLightHasher, g.Referent, g.targetPackage, g.importNamer)
	hp, ok := gg.(htrPutter)
	if !ok {
		return ""
	}
	return hp.generateHTRPutter(fieldName)
}

func (g *generatePointer) generateFixedMarshalValue(fieldName string) string {
	gg := newValueGenerator(interfaces.SszMarshaler, g.Referent, g.targetPackage, g.importNamer)
	return gg.generateFixedMarshalValue(fieldName)
}

func (g *generatePointer) generateUnmarshalValue(fieldName string, sliceName string) string {
	gg := newValueGenerator(interfaces.SszUnmarshaler, g.Referent, g.targetPackage, g.importNamer)
	return gg.generateUnmarshalValue(fieldName, sliceName)
}

func (g *generatePointer) initializeValue() string {
	gg := newValueGenerator(interfaces.SszMarshaler, g.Referent, g.targetPackage, g.importNamer)
	iv, ok := gg.(valueInitializer)
	if ok {
		return iv.initializeValue()
	}
	return ""
}

func (g *generatePointer) generateVariableMarshalValue(fieldName string) string {
	gg := newValueGenerator(interfaces.SszMarshaler, g.Referent, g.targetPackage, g.importNamer)
	vm, ok := gg.(variableMarshaller)
	if !ok {
		panic(fmt.Sprintf("variable size type does not implement variableMarshaller: %v", g.Referent))
	}
	return vm.generateVariableMarshalValue(fieldName)
}

func (g *generatePointer) variableSizeSSZ(fieldName string) string {
	gg := newValueGenerator(interfaces.SszMarshaler, g.Referent, g.targetPackage, g.importNamer)
	return gg.variableSizeSSZ(fieldName)
}

var _ valueGenerator = &generatePointer{}
var _ htrPutter = &generatePointer{}
