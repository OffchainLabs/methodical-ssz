package specs

import (
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/spf13/afero"
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

	require.Equal(t, 2, len(sr.GoTypes()))
}

func TestHarnessYamlFull(t *testing.T) {
	t.Skip("Skipping this test since no prysm.yaml file is available")
	sr := loadPrysmRelations(t)
	require.Equal(t, "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1", sr.Package)
}

func TestRelationsAtFork(t *testing.T) {
	t.Skip("Skipping this test since no prysm.yaml file is available")
	cases := []struct {
		name     string
		specName string
		typeName string
		fork     Fork
		err      error
		missing  bool
	}{
		{
			name:     "BeaconBlock at phase0",
			specName: "BeaconBlock",
			typeName: "BeaconBlock",
			fork:     Phase0,
		},
		{
			name:     "BeaconBlock at altair",
			specName: "BeaconBlock",
			typeName: "BeaconBlockAltair",
			fork:     Altair,
		},
		{
			name:     "Checkpoint at phase0",
			specName: "Checkpoint",
			typeName: "Checkpoint",
			fork:     Phase0,
		},
		{
			name:     "Checkpoint at altair",
			specName: "Checkpoint",
			typeName: "Checkpoint",
			fork:     Altair,
		},
		{
			name:     "Checkpoint at bellatrix",
			specName: "Checkpoint",
			typeName: "Checkpoint",
			fork:     Bellatrix,
		},
		{
			name:     "ExecutionPayload missing at phase0",
			specName: "ExecutionPayload",
			typeName: "",
			fork:     Phase0,
			missing:  true,
		},
		{
			name:     "ExecutionPayload missing at altair",
			specName: "ExecutionPayload",
			typeName: "",
			fork:     Altair,
			missing:  true,
		},
		{
			name:     "ExecutionPayload at bellatrix",
			specName: "ExecutionPayload",
			typeName: "ExecutionPayload",
			fork:     Bellatrix,
		},
		{
			name:     "ExecutionPayload at capella",
			specName: "ExecutionPayload",
			typeName: "ExecutionPayloadCapella",
			fork:     Capella,
		},
	}
	sr := loadPrysmRelations(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := sr.RelationsAtFork(c.fork)
			if err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			tn, ok := r[c.specName]
			if c.missing {
				require.Equal(t, false, ok)
				return
			}
			require.Equal(t, true, ok)
			require.Equal(t, c.typeName, tn)
		})
	}
}

func loadPrysmRelations(t *testing.T) *SpecRelationships {
	y, err := os.ReadFile("testdata/prysm.yaml")
	require.NoError(t, err)
	sr := &SpecRelationships{}
	err = yaml.Unmarshal([]byte(y), sr)
	require.NoError(t, err)
	return sr
}

func TestTestCaseTplFuncName(t *testing.T) {
	cases := []struct {
		name       string
		ident      TestIdent
		structname string
	}{
		{
			name: "Test_mainnet_altair_AggregateAndProof_0",
			ident: TestIdent{
				Preset: Mainnet,
				Fork:   Altair,
				Name:   "AggregateAndProof",
				Offset: 0,
			},
			structname: "AggregateAttestationAndProof",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tpl := TestCaseTpl{
				ident:      c.ident,
				structName: c.ident.Name,
			}
			require.Equal(t, c.name, tpl.TestFuncName())
		})
	}
}

func basicFixture() Fixture {
	return Fixture{
		Root:      FixtureFile{Contents: []byte(`{root: '0x44de62c118d7951f5b6d9a03444e54aff47d02ff57add2a4eb2a198b3e83ae35'}`)},
		Directory: "tests/mainnet/altair/ssz_static/AggregateAndProof/ssz_random/case_0",
	}
}

func TestCaseFileLayout(t *testing.T) {
	t.Skip("Skipping this test since no prysm.yaml file is available")
	fs := afero.NewMemMapFs()
	fix := basicFixture()
	cases := map[TestIdent]Fixture{
		TestIdent{Preset: Mainnet, Fork: Altair, Name: "Checkpoint", Offset: 0}: fix,
	}
	rels := &SpecRelationships{
		Package: "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1",
		Preset:  Mainnet,
		Defs: []ForkTypeDefinitions{
			{
				Fork: Altair,
				Types: []TypeRelation{
					{
						SpecName: "Checkpoint",
					},
				},
			},
		},
	}
	require.NoError(t, WriteSpecTestFiles(cases, rels, fs))
	entries, err := afero.ReadDir(fs, fix.Directory)
	require.NoError(t, err)
	searching := map[string]bool{
		rootFilename:       true,
		serializedFilename: true,
		valueFilename:      true,
	}
	for _, f := range entries {
		_, n, err := ParsePath(path.Join(fix.Directory, f.Name()))
		require.NoError(t, err)
		_, ok := searching[n]
		if ok {
			delete(searching, n)
		}
	}
	require.Equal(t, 0, len(searching))
}

func TestRenderTestCaseTpl(t *testing.T) {
	tpl := TestCaseTpl{
		ident: TestIdent{
			Preset: Mainnet,
			Fork:   Altair,
			Offset: 0,
		},
		fixture:    basicFixture(),
		structName: "AggregateAttestationAndProof",
	}
	rendered, err := tpl.Render()
	require.NoError(t, err)
	expected := `func Test_mainnet_altair_AggregateAttestationAndProof_0(t *testing.T) {
	fixtureDir := "testdata/tests/mainnet/altair/ssz_static/AggregateAndProof/ssz_random/case_0"
	root, serialized, err := specs.RootAndSerializedFromFixture(fixtureDir)
	require.NoError(t, err)
	v := &AggregateAttestationAndProof{}
	require.NoError(t, v.UnmarshalSSZ(serialized))
	sroot, err := v.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, root, sroot)
}`
	require.Equal(t, expected, rendered)
}
