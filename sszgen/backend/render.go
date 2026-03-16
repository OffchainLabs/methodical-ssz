package backend

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"strings"
	"text/template"

	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
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
	return strings.Join(gc.blocks, "\n\n")
}

func (gc *generatedCode) merge(right *generatedCode) {
	gc.blocks = append(gc.blocks, right.blocks...)
}

// Generator needs to be initialized with the package name,
// so use the new NewGenerator func for proper setup.
type Generator struct {
	gc          []*generatedCode
	packagePath string
	importNamer *ImportNamer
}

var defaultSSZImports = map[string]string{
	"github.com/prysmaticlabs/fastssz": "ssz",
	"fmt":                              "",
}

func NewGenerator(packagePath string) *Generator {
	importNamer := NewImportNamer(packagePath, defaultSSZImports)
	return &Generator{
		packagePath: packagePath,
		importNamer: importNamer,
	}
}

// TODO Generate should be able to return an error
func (g *Generator) Generate(vr gentypes.ValRep) error {
	if vc, ok := vr.(*gentypes.ValueContainer); ok {
		return g.genValueContainer(vc)
	}
	if vo, ok := vr.(*gentypes.ValueOverlay); ok {
		return g.genValueOverlay(vo)
	}
	return fmt.Errorf("can only generate method sets for container & overlay gentypes at this time, type: %v", vr.TypeName())
}

func (g *Generator) genValueOverlay(vc *gentypes.ValueOverlay) error {
	// TODO (MariusVanDerWijden) implement for basic gentypes
	return nil
}

func (g *Generator) genValueContainer(vc *gentypes.ValueContainer) error {
	container := &generateContainer{ValueContainer: vc, targetPackage: g.packagePath, importNamer: g.importNamer}
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
	ft := template.New("generated.ssz.go")
	tmpl, err := ft.Parse(fileTemplate)
	if err != nil {
		return nil, err
	}
	final := &generatedCode{}
	for _, gc := range g.gc {
		final.merge(gc)
	}
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, struct {
		Package string
		Imports string
		Blocks  string
	}{
		Package: RenderedPackageName(g.packagePath),
		Imports: g.importNamer.ImportPairs(),
		Blocks:  final.renderBlocks(),
	})
	if err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
	//return buf.Bytes(), nil
}

type valueGenerator interface {
	variableSizeSSZ(fieldname string) string
	generateFixedMarshalValue(string) string
	generateUnmarshalValue(string, string) string
	generateHTRPutter(string) string
}

type valueInitializer interface {
	initializeValue() string
}

type variableMarshaller interface {
	generateVariableMarshalValue(string) string
}

type coercer interface {
	coerce() func(string) string
}

type htrPutter interface {
	generateHTRPutter(string) string
}

func newValueGenerator(ifaceCtx *types.Interface, vr gentypes.ValRep, packagePath string, inm *ImportNamer) valueGenerator {
	if vr.SatisfiesInterface(ifaceCtx) {
		return &generateDelegate{ValRep: vr, targetPackage: packagePath, importNamer: inm}
	}
	switch ty := vr.(type) {
	case *gentypes.ValueBool:
		return &generateBool{valRep: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueByte:
		return &generateByte{ValueByte: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueContainer:
		return &generateContainer{ValueContainer: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueList:
		return &generateList{valRep: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueOverlay:
		return &generateOverlay{ValueOverlay: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValuePointer:
		return &generatePointer{ValuePointer: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueUint:
		return &generateUint{valRep: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueUnion:
		return &generateUnion{ValueUnion: ty, targetPackage: packagePath, importNamer: inm}
	case *gentypes.ValueVector:
		return &generateVector{valRep: ty, targetPackage: packagePath, importNamer: inm}
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

func fullyQualifiedTypeName(v gentypes.ValRep, targetPackage string, inamer *ImportNamer) string {
	tn := v.TypeName()
	if targetPackage == v.PackagePath() || v.PackagePath() == "" {
		return tn
	}
	pkg := inamer.NameString(v.PackagePath())

	if tn[0:1] == "*" {
		tn = tn[1:]
		pkg = "*" + pkg
	}
	return pkg + "." + tn
}

// RenderedPackageName reduces the fully qualified package name to the relative package name, ie
// github.com/prysmaticlabs/prysm/v3/proto/eth/v1 -> v1
func RenderedPackageName(n string) string {
	parts := strings.Split(n, "/")
	return parts[len(parts)-1]
}
