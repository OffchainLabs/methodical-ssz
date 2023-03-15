package backend

import (
	"fmt"

	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// "Delegate" meaning a type that defines it's own ssz methodset
type generateDelegate struct {
	types.ValRep
	targetPackage string
}

func (g *generateDelegate) generateHTRPutter(fieldName string) string {
	fullTmpl := `if err := %s.HashTreeRootWith(hh); err != nil {
		return err
	}`
	lightTmpl := `if hash, err := %s.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}`
	if g.SatisfiesInterface(interfaces.SszFullHasher) {
		return fmt.Sprintf(fullTmpl, fieldName)
	}
	return fmt.Sprintf(lightTmpl, fieldName)
}

func (g *generateDelegate) variableSizeSSZ(fieldName string) string {
	return fmt.Sprintf("%s.SizeSSZ()", fieldName)
}

func (g *generateDelegate) generateUnmarshalValue(fieldName string, sliceName string) string {
	t := `if err = %s.UnmarshalSSZ(%s); err != nil {
		return err
	}`
	return fmt.Sprintf(t, fieldName, sliceName)
}

func (g *generateDelegate) generateFixedMarshalValue(fieldName string) string {
	if g.IsVariableSized() {
		return fmt.Sprintf(`dst = ssz.WriteOffset(dst, offset)
offset += %s.SizeSSZ()`, fieldName)
	}
	return g.generateDelegateFieldMarshalSSZ(fieldName)
}

// method that generates code which calls the MarshalSSZ method of the field
func (g *generateDelegate) generateDelegateFieldMarshalSSZ(fieldName string) string {
	return fmt.Sprintf(`if dst, err = %s.MarshalSSZTo(dst); err != nil {
		return nil, err
	}`, fieldName)
}

func (g *generateDelegate) generateVariableMarshalValue(fieldName string) string {
	return g.generateDelegateFieldMarshalSSZ(fieldName)
}

func (g *generateDelegate) initializeValue() string {
	return initializeValue(g.ValRep, g.targetPackage)
}

var _ valueGenerator = &generateDelegate{}
var _ htrPutter = &generateDelegate{}
var _ valueInitializer = &generateDelegate{}
