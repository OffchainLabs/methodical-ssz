package backend

import (
	"go/format"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
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
	require.NoError(t, err)
	require.Equal(t, generator_generateFixture, string(rendered))
}

func TestGenerator_GenerateBeaconState(t *testing.T) {
	t.Skip("fixtures need to be updated")
	b, err := os.ReadFile("testdata/TestGenerator_GenerateBeaconState.expected")
	require.NoError(t, err)
	formatted, err := format.Source(b)
	require.NoError(t, err)
	expected := string(formatted)

	g := &Generator{
		packagePath: "github.com/prysmaticlabs/prysm/v3/proto/beacon/p2p/v1",
	}
	g.Generate(testFixBeaconState)
	rendered, err := g.Render()
	require.NoError(t, err)
	actual := string(rendered)
	require.Equal(t, expected, actual)
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
		require.Equal(t, importAlias(c.packageName), c.alias)
	}
}

func TestRenderedPackageName(t *testing.T) {
	before := "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	after := "v1"
	require.Equal(t, after, RenderedPackageName(before))
}
