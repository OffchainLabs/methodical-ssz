package core

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"
)

// DefaultSSZImports is seeded into the ImportNamer for a generated SSZ file.
// Seeding claims the identifiers up front so alias assignment is stable, but an
// entry is only emitted if the rendered body actually references it.
var DefaultSSZImports = map[string]string{
	"github.com/OffchainLabs/methodical-ssz/ssz": "ssz",
	"fmt":             "",
	"encoding/binary": "binary", // explicit alias kept only for golden-test stability
}

var fileTemplate = template.Must(template.New("generated.ssz.go").Parse(
	`package {{.Package}}

{{ if .Imports -}}
import (
	{{.Imports}}
)
{{- end }}

{{.Blocks}}`))

// RenderFile assembles a single Go source file from a package name, a shared
// ImportNamer (already populated during generation), and one or more groups of
// code blocks. Block groups are concatenated in order, separated by blank lines,
// then gofmt-formatted. Sharing one namer + one RenderFile across multiple
// generators is how their methods land in a single import-consistent file.
func RenderFile(pkgName string, namer *ImportNamer, blockGroups ...[]string) ([]byte, error) {
	if pkgName == "" {
		return nil, fmt.Errorf("RenderFile requires a package name")
	}
	blocks := make([]string, 0)
	for _, g := range blockGroups {
		blocks = append(blocks, g...)
	}

	body := strings.Join(blocks, "\n\n")
	namer.PruneUnreferenced(body)

	buf := bytes.NewBuffer(nil)
	err := fileTemplate.Execute(buf, struct {
		Package string
		Imports string
		Blocks  string
	}{
		Package: pkgName,
		Imports: namer.ImportPairs(),
		Blocks:  body,
	})
	if err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

// RenderedPackageName reduces the fully qualified package name to the relative package name, ie
// github.com/prysmaticlabs/prysm/v3/proto/eth/v1 -> v1
func RenderedPackageName(n string) string {
	parts := strings.Split(n, "/")
	return parts[len(parts)-1]
}
