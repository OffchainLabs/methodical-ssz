package config

import (
	"errors"
	"fmt"

	"sigs.k8s.io/yaml"
)

var errNoTypes = errors.New("no types specified in generator config")
var errNoPackage = errors.New("package not specified in generator config")

// maxActiveFields is the spec bound: a ProgressiveContainer's active_fields
// configuration is restricted to ≤ 256 bits.
const maxActiveFields = 256

// ProgressiveConfig marks a type as an SSZ ProgressiveContainer. An empty
// object means every position in the active-fields bitvector is active (one
// per Go struct field). InactiveIndices lists 0-indexed positions in the
// bitvector forced to 0 (reserved/removed fields); the bitvector's final
// length is the Go field count plus the number of inactive indices.
type ProgressiveConfig struct {
	InactiveIndices []int `json:"inactive_indices"`
}

// ActiveFields expands the configuration into the spec's active_fields
// bitvector for a container with numFields declared Go fields, validating the
// spec's legality rules.
func (p *ProgressiveConfig) ActiveFields(numFields int) ([]bool, error) {
	if numFields == 0 {
		return nil, errors.New("progressive containers with no fields are illegal")
	}
	length := numFields + len(p.InactiveIndices)
	if length > maxActiveFields {
		return nil, fmt.Errorf("active_fields is restricted to %d bits, got %d", maxActiveFields, length)
	}
	af := make([]bool, length)
	for i := range af {
		af[i] = true
	}
	for _, idx := range p.InactiveIndices {
		if idx < 0 || idx >= length {
			return nil, fmt.Errorf("inactive index %d out of range for active_fields of length %d", idx, length)
		}
		if !af[idx] {
			return nil, fmt.Errorf("duplicate inactive index %d", idx)
		}
		af[idx] = false
	}
	if !af[length-1] {
		return nil, errors.New("active_fields configurations ending in 0 are illegal")
	}
	return af, nil
}

// FieldConfig overrides the SSZ interpretation of a single Go struct field.
// Type may be ProgressiveList, ProgressiveBitlist, or ProgressiveByteList.
// Element optionally overrides the interpretation of the field's element type,
// for nested progressive collections that have no named Go type to delegate to
// (e.g. a ProgressiveList whose elements are themselves ProgressiveByteLists:
// `{type: "ProgressiveList", element: {type: "ProgressiveByteList"}}`).
type FieldConfig struct {
	Type    string       `json:"type"`
	Element *FieldConfig `json:"element,omitempty"`
}

const (
	FieldTypeProgressiveList     = "ProgressiveList"
	FieldTypeProgressiveBitlist  = "ProgressiveBitlist"
	FieldTypeProgressiveByteList = "ProgressiveByteList"
)

// isProgressiveFieldType reports whether a FieldConfig.Type names a progressive
// collection.
func isProgressiveFieldType(t string) bool {
	switch t {
	case FieldTypeProgressiveList, FieldTypeProgressiveBitlist, FieldTypeProgressiveByteList:
		return true
	default:
		return false
	}
}

// TypeConfig is a struct that represents a type to generate.
type TypeConfig struct {
	Name string `json:"name"`
	// Progressive marks the type as an SSZ ProgressiveContainer.
	Progressive *ProgressiveConfig `json:"progressive"`
	// Fields overrides the SSZ interpretation of individual Go struct fields,
	// keyed by field name.
	Fields map[string]FieldConfig `json:"fields"`
}

// GeneratorConfig is an alternative to specifying the package and type to generate as cli args.
// This allows deeper customization of code gen, such as marking types as
// progressive containers or fields as progressive collections.
type GeneratorConfig struct {
	Package string       `json:"package"`
	Types   []TypeConfig `json:"types"`
	TypeMap map[string]TypeConfig
}

func (g *GeneratorConfig) populateTypeMap() {
	g.TypeMap = make(map[string]TypeConfig)
	for _, t := range g.Types {
		g.TypeMap[t.Name] = t
	}
}

func (g *GeneratorConfig) TypeNames() []string {
	names := make([]string, len(g.Types))
	for i := range g.Types {
		names[i] = g.Types[i].Name
	}
	return names
}

func (g *GeneratorConfig) TypeConfig(name string) TypeConfig {
	if t, ok := g.TypeMap[name]; ok {
		return t
	}
	return TypeConfig{}
}

// ParseGeneratorConfig parses a json/yaml config file into a GeneratorConfig struct.
func ParseGeneratorConfig(input []byte) (*GeneratorConfig, error) {
	g := &GeneratorConfig{}
	if err := yaml.Unmarshal(input, g); err != nil {
		return nil, err
	}
	if len(g.Types) == 0 {
		return nil, errNoTypes
	}
	if len(g.Package) == 0 {
		return nil, errNoPackage
	}
	g.populateTypeMap()
	return g, nil
}

// DisableProgressive strips every progressive attribute from the config (the
// container marker and per-field ProgressiveList/Bitlist/ByteList overrides) so
// codegen produces standard SSZ. Backs the generate command's
// --disable-progressive flag.
func (g *GeneratorConfig) DisableProgressive() {
	for i := range g.Types {
		g.Types[i].Progressive = nil
		for name, fc := range g.Types[i].Fields {
			if isProgressiveFieldType(fc.Type) {
				delete(g.Types[i].Fields, name)
			}
		}
	}
	g.populateTypeMap()
}

// NewGeneratorConfig can be used to create a GeneratorConfig struct with a package and list of types.
func NewGeneratorConfig(pkg string, typeNames []string) (*GeneratorConfig, error) {
	if pkg == "" {
		return nil, errNoPackage
	}
	if len(typeNames) == 0 {
		return nil, errNoTypes
	}
	g := &GeneratorConfig{Package: pkg, Types: make([]TypeConfig, len(typeNames))}
	for i := range typeNames {
		g.Types[i] = TypeConfig{Name: typeNames[i]}
	}
	g.populateTypeMap()
	return g, nil
}
