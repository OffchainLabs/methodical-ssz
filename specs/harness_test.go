package specs

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"sigs.k8s.io/yaml"
)

func TestHarnessYaml(t *testing.T) {
	input := `
package: github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1
preset: mainnet
defs:
  - fork: phase0
    types:
      - name: BeaconBlock
  - fork: altair
    types:
      - name: BeaconBlock
        type_name: BeaconBlockAltair`
	sr := &SpecRelationships{}
	err := yaml.Unmarshal([]byte(input), sr)
	require.NoError(t, err)
	require.Equal(t, "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1", sr.Package)
	require.Equal(t, Mainnet, sr.Preset)
	require.Equal(t, 2, len(sr.Defs))
	require.Equal(t, Phase0, sr.Defs[0].Fork)
	require.Equal(t, 1, len(sr.Defs[0].Types))
	require.Equal(t, "BeaconBlock", sr.Defs[0].Types[0].SpecName)
	require.Equal(t, "", sr.Defs[0].Types[0].TypeName)
	require.Equal(t, Altair, sr.Defs[1].Fork)
	require.Equal(t, 1, len(sr.Defs[1].Types))
	require.Equal(t, "BeaconBlock", sr.Defs[1].Types[0].SpecName)
	require.Equal(t, "BeaconBlockAltair", sr.Defs[1].Types[0].TypeName)
}