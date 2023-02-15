package backend

import (
	"go/format"
	"os"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
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
		imports: map[string]string{
			"github.com/prysmaticlabs/derp/derp": "derp",
			"github.com/prysmaticlabs/fastssz":   "ssz",
			"fmt":                                "",
		},
	}
	g := &Generator{packagePath: "github.com/prysmaticlabs/derp", packageName: "derp"}
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
		packageName: "ethereum_beacon_p2p_v1",
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

func TestExtractImportsFromContainerFields(t *testing.T) {
	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	targetPackage := "github.com/prysmaticlabs/prysm/v3/proto/beacon/p2p/v1"
	imports := extractImportsFromContainerFields(vc.Contents, targetPackage)
	require.Equal(t, 3, len(imports))
	require.Equal(t, "prysmaticlabs_eth2_types", imports["github.com/prysmaticlabs/eth2-types"])
	require.Equal(t, "prysmaticlabs_prysm_v3_proto_eth_v1alpha1", imports["github.com/prysmaticlabs/prysm/v3/proto/eth/v1alpha1"])
	require.Equal(t, "prysmaticlabs_go_bitfield", imports["github.com/prysmaticlabs/go-bitfield"])
}

func TestRenderedPackageName(t *testing.T) {
	before := "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	after := "v1"
	require.Equal(t, after, RenderedPackageName(before))
}
