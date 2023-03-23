package backend

import (
	"fmt"
	"go/types"
	"strings"
)

type ImportNamer struct {
	source  string
	aliases map[string]string // map from package path -> alias
	reverse map[string]string //  map from alias -> package path
}

var pbRuntime = "google.golang.org/protobuf/runtime/protoimpl"
var protoAliases = map[string]string{
	pbRuntime: "protoimpl",
	"google.golang.org/protobuf/internal/impl": "protoimpl",
}

func (n *ImportNamer) NameString(p string) string {
	if alias, isProto := protoAliases[p]; isProto {
		n.reverse[alias] = pbRuntime
		n.aliases[pbRuntime] = alias
		return alias
	}
	// no package name for self
	if p == n.source {
		return ""
	}
	name, exists := n.aliases[p]
	if exists {
		return name
	}
	// build increasingly long path suffixes until a unique one is found
	parts := strings.Split(p, "/")
	for i := 0; i < len(parts); i++ {
		name := strings.Join(parts[len(parts)-1-i:], "_")
		// deal with domain portion of path for extreme case where 2 packages only differ in domain
		name = strings.ReplaceAll(name, ".", "_")
		// dashes are valid in package names but not go identifiers - like go-bitfield
		name = strings.ReplaceAll(name, "-", "_")
		_, conflict := n.reverse[name]
		if conflict {
			continue
		}
		n.reverse[name] = p
		n.aliases[p] = name
		return name
	}
	panic(fmt.Sprintf("unable to find unique name for package %s", p))
}

func (n *ImportNamer) Name(p *types.Package) string {
	return n.NameString(p.Path())
}

func (n *ImportNamer) ImportSource() string {
	imports := make([]string, 0)
	for pkg, alias := range n.aliases {
		if alias == "google.golang.org/protobuf/internal/impl" {
			imports = append(imports, fmt.Sprintf("%s \"google.golang.org/protobuf/runtime/protoimpl\"", alias))
		} else {
			imports = append(imports, fmt.Sprintf("%s \"%s\"", alias, pkg))
		}
	}

	return fmt.Sprintf("import (\n%s\n)\n", strings.Join(imports, "\n"))
}

func (n *ImportNamer) ImportPairs() string {
	imports := make([]string, 0)
	stdImports := make([]string, 0)
	for alias, pkg := range n.reverse {
		if pkg == "" {
			stdImports = append(stdImports, fmt.Sprintf("\"%s\"", alias))
		} else {
			imports = append(imports, fmt.Sprintf("%s \"%s\"", alias, pkg))
		}
	}

	if len(stdImports) > 0 {
		return strings.Join(stdImports, "\n") + "\n\n" + strings.Join(imports, "\n")
	}
	return strings.Join(imports, "\n")
}

func NewImportNamer(source string, defaults map[string]string) *ImportNamer {
	aliases := make(map[string]string)
	reverse := make(map[string]string)
	for pkg, alias := range defaults {
		if pkg != "" {
			aliases[pkg] = alias
		}
		reverse[alias] = pkg
	}
	return &ImportNamer{
		source:  source,
		aliases: aliases,
		reverse: reverse,
	}
}
