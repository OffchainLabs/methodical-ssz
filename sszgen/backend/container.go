package backend

import (
	"fmt"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

const receiverName = "c"

type generateContainer struct {
	*types.ValueContainer
	targetPackage string
}

func (g *generateContainer) generateHTRPutter(fieldName string) string {
	fullTmpl := `if err := %s.HashTreeRootWith(hh); err != nil {
		return err
	}`
	lightTmpl := `if hash, err := %s.HashTreeRoot(); err != nil {
		return err
	} else {
		hh.AppendBytes32(hash[:])
	}`
	if !g.LightHash {
		return fmt.Sprintf(fullTmpl, fieldName)
	}
	return fmt.Sprintf(lightTmpl, fieldName)
}

func (g *generateContainer) variableSizeSSZ(fieldName string) string {
	return fmt.Sprintf("%s.SizeSSZ()", fieldName)
}

func (g *generateContainer) generateUnmarshalValue(fieldName string, sliceName string) string {
	t := `if err = %s.UnmarshalSSZ(%s); err != nil {
		return err
	}`
	return fmt.Sprintf(t, fieldName, sliceName)
}

func (g *generateContainer) generateFixedMarshalValue(fieldName string) string {
	if g.IsVariableSized() {
		return fmt.Sprintf(`dst = ssz.WriteOffset(dst, offset)
offset += %s.SizeSSZ()`, fieldName)
	}
	return g.generateDelegateFieldMarshalSSZ(fieldName)
}

// method that generates code which calls the MarshalSSZ method of the field
func (g *generateContainer) generateDelegateFieldMarshalSSZ(fieldName string) string {
	return fmt.Sprintf(`if dst, err = %s.MarshalSSZTo(dst); err != nil {
		return nil, err
	}`, fieldName)
}

func (g *generateContainer) generateVariableMarshalValue(fieldName string) string {
	return g.generateDelegateFieldMarshalSSZ(fieldName)
}

func (g *generateContainer) fixedOffset() int {
	offset := 0
	for _, c := range g.Contents {
		offset += c.Value.FixedSize()
	}
	return offset
}

func (g *generateContainer) initializeValue(fieldName string) string {
	if g.Value {
		return ""
	}
	return fmt.Sprintf("new(%s)", fullyQualifiedTypeName(g.ValueContainer, g.targetPackage))
}

var _ valueGenerator = &generateContainer{}
var _ valueInitializer = &generateContainer{}
var _ htrPutter = &generateContainer{}
