package backend

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

type generatedCode struct {
	blocks []string
	// key=package path, value=alias
	imports map[string]string
}

func (gc *generatedCode) renderImportPairs() string {
	pairs := make([]string, 0)
	for k, v := range gc.imports {
		pairs = append(pairs, fmt.Sprintf("%s \"%s\"", v, k))
	}
	return strings.Join(pairs, "\n")
}

func (gc *generatedCode) renderBlocks() string {
	return strings.Join(gc.blocks, "\n")
}

func (gc *generatedCode) merge(right *generatedCode) {
	gc.blocks = append(gc.blocks, right.blocks...)
	if right.imports == nil {
		return
	}
	for k, v := range right.imports {
		// deduplicate imports and detect collisions
		// we should prevent collisions by normalizing import naming in a preprocessing pass
		if _, ok := gc.imports[k]; ok {
			continue
		}
		gc.imports[k] = v
	}
}

// Generator needs to be initialized with the package name,
// so use the new NewGenerator func for proper setup.
type Generator struct {
	gc          []*generatedCode
	packageName string
	packagePath string
}

func NewGenerator(packageName, packagePath string) *Generator {
	return &Generator{
		packageName: packageName,
		packagePath: packagePath,
	}
}

// TODO Generate should be able to return an error
func (g *Generator) Generate(vr types.ValRep) error {
	if vc, ok := vr.(*types.ValueContainer); ok {
		return g.genValueContainer(vc)
	}
	if vo, ok := vr.(*types.ValueOverlay); ok {
		return g.genValueOverlay(vo)
	}
	return fmt.Errorf("can only generate method sets for container & overlay types at this time, type: %v", vr.TypeName())
}

func (g *Generator) genValueOverlay(vc *types.ValueOverlay) error {
	// TODO (MariusVanDerWijden) implement for basic types
	return nil
}

func (g *Generator) genValueContainer(vc *types.ValueContainer) error {
	container := &generateContainer{vc, g.packagePath}
	methods := []func(gc *generateContainer) (*generatedCode, error){
		GenerateSizeSSZ,
		GenerateMarshalSSZ,
		GenerateUnmarshalSSZ,
		GenerateHashTreeRoot,
	}
	for _, method := range methods {
		code, err := method(container)
		if err != nil {
			return err
		}
		g.gc = append(g.gc, code)
	}
	return nil
}

var fileTemplate = `package {{.Package}}

{{ if .Imports -}}
import (
	{{.Imports}}
)
{{- end }}

{{.Blocks}}`

func (g *Generator) Render() ([]byte, error) {
	if g.packagePath == "" {
		return nil, fmt.Errorf("missing packagePath: Generator requires a packagePath for code generation.")
	}
	if g.packageName == "" {
		return nil, fmt.Errorf("missing packageName: Generator requires a target package name for code generation.")
	}
	ft := template.New("generated.ssz.go")
	tmpl, err := ft.Parse(fileTemplate)
	if err != nil {
		return nil, err
	}
	final := &generatedCode{
		imports: map[string]string{
			"github.com/prysmaticlabs/fastssz": "ssz",
			"fmt":                              "",
		},
	}
	for _, gc := range g.gc {
		final.merge(gc)
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, struct {
		Package string
		Imports string
		Blocks  string
	}{
		Package: RenderedPackageName(g.packageName),
		Imports: final.renderImportPairs(),
		Blocks:  final.renderBlocks(),
	})
	if err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

type valueGenerator interface {
	variableSizeSSZ(fieldname string) string
	generateFixedMarshalValue(string) string
	generateUnmarshalValue(string, string) string
	generateHTRPutter(string) string
}

type valueInitializer interface {
	initializeValue(string) string
}

type variableMarshaller interface {
	generateVariableMarshalValue(string) string
}

type variableUnmarshaller interface {
	generateVariableUnmarshalValue(string) string
}

type coercer interface {
	coerce() func(string) string
}

type htrPutter interface {
	generateHTRPutter(string) string
}

func newValueGenerator(vr types.ValRep, packagePath string) valueGenerator {
	switch ty := vr.(type) {
	case *types.ValueBool:
		return &generateBool{valRep: ty, targetPackage: packagePath}
	case *types.ValueByte:
		return &generateByte{ty, packagePath}
	case *types.ValueContainer:
		return &generateContainer{ty, packagePath}
	case *types.ValueList:
		return &generateList{valRep: ty, targetPackage: packagePath}
	case *types.ValueOverlay:
		return &generateOverlay{ty, packagePath}
	case *types.ValuePointer:
		return &generatePointer{ty, packagePath}
	case *types.ValueUint:
		return &generateUint{valRep: ty, targetPackage: packagePath}
	case *types.ValueUnion:
		return &generateUnion{ty, packagePath}
	case *types.ValueVector:
		return &generateVector{valRep: ty, targetPackage: packagePath}
	}
	panic(fmt.Sprintf("Cannot manage generation for unrecognized ValRep implementation %v", vr))
}

func importAlias(packageName string) string {
	parts := strings.Split(packageName, "/")
	for i, p := range parts {
		if strings.Contains(p, ".") {
			continue
		}
		parts = parts[i:]
		break
	}
	return strings.ReplaceAll(strings.Join(parts, "_"), "-", "_")
}

func fullyQualifiedTypeName(v types.ValRep, targetPackage string) string {
	tn := v.TypeName()
	if targetPackage == v.PackagePath() || v.PackagePath() == "" {
		return tn
	}
	parts := strings.Split(v.PackagePath(), "/")
	for i, p := range parts {
		if strings.Contains(p, ".") {
			continue
		}
		parts = parts[i:]
		break
	}
	pkg := strings.ReplaceAll(strings.Join(parts, "_"), "-", "_")
	if tn[0:1] == "*" {
		tn = tn[1:]
		pkg = "*" + pkg
	}
	return pkg + "." + tn
}

func extractImportsFromContainerFields(cfs []types.ContainerField, targetPackage string) map[string]string {
	imports := make(map[string]string)
	for _, cf := range cfs {
		pkg := cf.Value.PackagePath()
		if pkg == "" || pkg == targetPackage {
			continue
		}
		imports[pkg] = importAlias(pkg)
	}
	return imports
}

// RenderedPackageName reduces the fully qualified package name to the relative package name, ie
// github.com/prysmaticlabs/prysm/v3/proto/eth/v1 -> v1
func RenderedPackageName(n string) string {
	parts := strings.Split(n, "/")
	return parts[len(parts)-1]
}
