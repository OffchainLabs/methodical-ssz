package types

import (
	"fmt"
	"go/types"
)

type ContainerField struct {
	Key   string
	Value ValRep
}

type ValueContainer struct {
	Name          string
	Package       string
	Contents      []ContainerField
	HasSSZMethods bool
	nameMap       map[string]ValRep
	Interfaces    map[*types.Interface]bool
}

func (vc *ValueContainer) Fields() []ContainerField {
	return vc.Contents
}

func (vc *ValueContainer) Append(name string, value ValRep) {
	vc.Contents = append(vc.Contents, ContainerField{Key: name, Value: value})
	if vc.nameMap == nil {
		vc.nameMap = make(map[string]ValRep)
	}
	vc.nameMap[name] = value
}

func (vc *ValueContainer) GetField(name string) (ValRep, error) {
	field, ok := vc.nameMap[name]
	if !ok {
		return nil, fmt.Errorf("Field named %s not found in container value mapping", name)
	}
	return field, nil
}

func (vc *ValueContainer) TypeName() string {
	return vc.Name
}

func (vc *ValueContainer) PackagePath() string {
	return vc.Package
}

func (vc *ValueContainer) FixedSize() int {
	if vc.IsVariableSized() {
		return 4
	}
	total := 0
	for _, c := range vc.Contents {
		o := c.Value
		total += o.FixedSize()
	}
	return total
}

func (vc *ValueContainer) IsVariableSized() bool {
	for _, c := range vc.Contents {
		if c.Value.IsVariableSized() {
			return true
		}
	}
	return false
}

func (vc *ValueContainer) SatisfiesInterface(ti *types.Interface) bool {
	return vc.Interfaces != nil && vc.Interfaces[ti]
}

var _ ValRep = &ValueContainer{}
