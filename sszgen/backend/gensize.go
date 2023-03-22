package backend

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
)

var sizeBodyTmpl = `func ({{.Receiver}} {{.Type}}) SizeSSZ() (int) {
	size := {{.FixedSize}}
	{{- .VariableSize }}
	return size
}`

func GenerateSizeSSZ(g *generateContainer) (*generatedCode, error) {
	sizeTmpl, err := template.New("GenerateSizeSSZ").Parse(sizeBodyTmpl)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)

	fixedSize := 0
	variableComputations := make([]string, 0)
	for _, c := range g.Contents {
		vg := newValueGenerator(interfaces.SszMarshaler, c.Value, g.targetPackage, g.importNamer)
		fixedSize += c.Value.FixedSize()
		if !c.Value.IsVariableSized() {
			continue
		}
		fieldName := fmt.Sprintf("%s.%s", receiverName, c.Key)
		vi, ok := vg.(valueInitializer)
		if ok {
			ini := vi.initializeValue()
			if ini != "" {
				variableComputations = append(variableComputations, fmt.Sprintf("if %s == nil {\n\t%s = %s\n}", fieldName, fieldName, ini))
			}
		}
		cv := vg.variableSizeSSZ(fieldName)
		if cv != "" {
			variableComputations = append(variableComputations, fmt.Sprintf("\tsize += %s", cv))
		}
	}

	err = sizeTmpl.Execute(buf, struct {
		Receiver     string
		Type         string
		FixedSize    int
		VariableSize string
	}{
		Receiver:     receiverName,
		Type:         fmt.Sprintf("*%s", g.TypeName()),
		FixedSize:    fixedSize,
		VariableSize: "\n" + strings.Join(variableComputations, "\n"),
	})
	if err != nil {
		return nil, err
	}
	return &generatedCode{
		blocks: []string{buf.String()},
	}, nil
}
