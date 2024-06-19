package backend

import (
	"go/format"
	"os"
	"testing"
)

var generator_generateFixture = `package derp

import (
	"fmt"
	derp "github.com/prysmaticlabs/derp/derp"
	ssz "github.com/prysmaticlabs/fastssz"
)

func main() {
	fmt.printf("hello world")
}
`

func TestGenerator_Generate(t *testing.T) {
	gc := &generatedCode{
		blocks: []string{"func main() {\n\tfmt.printf(\"hello world\")\n}"},
	}
	defaultImports := map[string]string{
		"github.com/prysmaticlabs/derp/derp": "derp",
		"github.com/prysmaticlabs/fastssz":   "ssz",
		"fmt":                                "",
	}
	inm := NewImportNamer("github.com/prysmaticlabs/derp", defaultImports)
	g := &Generator{packagePath: "github.com/prysmaticlabs/derp", importNamer: inm}
	g.gc = append(g.gc, gc)
	rendered, err := g.Render()
	if err != nil {
		t.Fatal(err)
	}
	if string(rendered) != generator_generateFixture {
		t.Fatalf("expected:\n%s\nactual:\n%s", generator_generateFixture, string(rendered))
	}
}

func TestGenerator_GenerateBeaconState(t *testing.T) {
	t.Skip("fixtures need to be updated")
	b, err := os.ReadFile("testdata/TestGenerator_GenerateBeaconState.expected")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := string(formatted)

	g := &Generator{
		packagePath: "github.com/prysmaticlabs/prysm/v3/proto/beacon/p2p/v1",
	}
	g.Generate(testFixBeaconState)
	rendered, err := g.Render()
	if err != nil {
		t.Fatal(err)
	}
	actual := string(rendered)
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestImportAlias(t *testing.T) {
	cases := []struct {
		packageName string
		alias       string
	}{
		{
			packageName: "github.com/derp/derp",
			alias:       "derp_derp",
		},
		{
			packageName: "text/template",
			alias:       "text_template",
		},
		{
			packageName: "fmt",
			alias:       "fmt",
		},
	}
	for _, c := range cases {
		if c.alias != importAlias(c.packageName) {
			t.Fatalf("unexpected importAlias for packageName %s, want=%s, got=%s", c.packageName, c.alias, importAlias(c.packageName))
		}
	}
}

func TestRenderedPackageName(t *testing.T) {
	before := "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	after := "v1"
	got := RenderedPackageName(before)
	if got != after {
		t.Fatalf("unexpected result for RenderedPackageName, want=%s, got=%s", after, got)
	}
}
