package spectest

import (
	"errors"
	"os"
	"path"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/specs"
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
	if err != nil {
		t.Fatalf("unexpected error from yaml.Unmarshal = %s", err.Error())
	}
	expectedPackage := "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	if sr.Package != expectedPackage {
		t.Errorf(".Package mismatch, want=%s, got=%s", expectedPackage, sr.Package)
	}
	if sr.Preset != specs.Mainnet {
		t.Errorf(".Preset mismatch, want=%s, got=%s", specs.Mainnet, sr.Preset)
	}
	if len(sr.Defs) != 2 {
		t.Errorf(".Defs mismatch, want=%d, got=%d", 2, len(sr.Defs))
	}
	if sr.Defs[0].Fork != specs.Phase0 {
		t.Errorf(".Defs[0].Fork mismatch, want=%s, got=%s", specs.Phase0, sr.Defs[0].Fork)
	}
	if len(sr.Defs[0].Types) != 1 {
		t.Errorf(".Defs[0].Types len mismatch, want=%d, got=%d", 1, len(sr.Defs[0].Types))
	}
	if sr.Defs[0].Types[0].SpecName != "BeaconBlock" {
		t.Errorf(".SpecName of first def wrong, want=%s, got=%s", "BeaconBlock", sr.Defs[0].Types[0].SpecName)
	}
	if sr.Defs[0].Types[0].TypeName != "" {
		t.Errorf(".TypeName of first type in first def wrong, want=%s, got=%s", "", sr.Defs[0].Types[0].TypeName)
	}
	if sr.Defs[1].Fork != specs.Altair {
		t.Errorf("wanted fork of 2nd def to be %s, got %s", specs.Altair, sr.Defs[1].Fork)
	}
	if len(sr.Defs[1].Types) != 1 {
		t.Errorf("wanted 1 type in the 2nd def, got %d", len(sr.Defs[1].Types))
	}
	if sr.Defs[1].Types[0].SpecName != "BeaconBlock" {
		t.Errorf("wrong .SpecName for the first type in the 2nd def, want=%s, got=%s", "BeaconBlock", sr.Defs[1].Types[0].SpecName)
	}
	if sr.Defs[1].Types[0].TypeName != "BeaconBlockAltair" {
		t.Errorf("wrong .TypeName fors the first type in the 2nd def, want=%s, got=%s", "BeaconBlockAltair", sr.Defs[1].Types[0].TypeName)
	}
	if len(sr.GoTypes()) != 2 {
		t.Errorf("wanted 2 go types overall, got %d", len(sr.GoTypes()))
	}
}

func TestHarnessYamlFull(t *testing.T) {
	sr := loadPrysmRelations(t)
	want := "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	if sr.Package != want {
		t.Fatalf("wanted .Package %s, got %s", want, sr.Package)
	}
}

func TestRelationsAtFork(t *testing.T) {
	cases := []struct {
		name     string
		specName string
		typeName string
		fork     specs.Fork
		err      error
		missing  bool
	}{
		{
			name:     "BeaconBlock at phase0",
			specName: "BeaconBlock",
			typeName: "BeaconBlock",
			fork:     specs.Phase0,
		},
		{
			name:     "BeaconBlock at altair",
			specName: "BeaconBlock",
			typeName: "BeaconBlockAltair",
			fork:     specs.Altair,
		},
		{
			name:     "Checkpoint at phase0",
			specName: "Checkpoint",
			typeName: "Checkpoint",
			fork:     specs.Phase0,
		},
		{
			name:     "Checkpoint at altair",
			specName: "Checkpoint",
			typeName: "Checkpoint",
			fork:     specs.Altair,
		},
		{
			name:     "Checkpoint at bellatrix",
			specName: "Checkpoint",
			typeName: "Checkpoint",
			fork:     specs.Bellatrix,
		},
		{
			name:     "ExecutionPayload missing at phase0",
			specName: "ExecutionPayload",
			typeName: "",
			fork:     specs.Phase0,
			missing:  true,
		},
		{
			name:     "ExecutionPayload missing at altair",
			specName: "ExecutionPayload",
			typeName: "",
			fork:     specs.Altair,
			missing:  true,
		},
		{
			name:     "ExecutionPayload at bellatrix",
			specName: "ExecutionPayload",
			typeName: "ExecutionPayload",
			fork:     specs.Bellatrix,
		},
		{
			name:     "ExecutionPayload at capella",
			specName: "ExecutionPayload",
			typeName: "ExecutionPayloadCapella",
			fork:     specs.Capella,
		},
	}
	sr := loadPrysmRelations(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := sr.RelationsAtFork(c.fork)
			if c.err != nil {
				if !errors.Is(err, c.err) {
					t.Fatal("wrong error type for RelationsAtFork")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error from sr.RelationsAtFork = %s", err.Error())
			}
			tn, ok := r[c.specName]
			if c.missing {
				if ok {
					t.Fatalf("spec name %s not missing as expected", c.specName)
				}
				return
			}
			if !ok {
				t.Fatalf("spec name %s missing", c.specName)
			}
			if tn.TypeName != c.typeName {
				t.Errorf("want relations type name %s, got %s", c.typeName, tn.TypeName)
			}
			// the prysm fixture sets no per-type packages, so every resolved
			// type defaults to the config's top-level package
			if tn.Package != sr.Package {
				t.Errorf("want default package %s, got %s", sr.Package, tn.Package)
			}
		})
	}
}

func loadPrysmRelations(t *testing.T) *SpecRelationships {
	fname := "testdata/prysm.yaml"
	y, err := os.ReadFile(fname)
	if err != nil {
		t.Fatalf("error reading file %s", fname)
	}
	sr := &SpecRelationships{}
	err = yaml.Unmarshal(y, sr)
	if err != nil {
		t.Fatalf("error unmarshaling yaml at %s", fname)
	}
	return sr
}

func TestTestCaseTplFuncName(t *testing.T) {
	cases := []struct {
		name       string
		ident      specs.TestIdent
		structname string
	}{
		{
			name: "Test_mainnet_altair_AggregateAndProof_0",
			ident: specs.TestIdent{
				Preset: specs.Mainnet,
				Fork:   specs.Altair,
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
			if tpl.TestFuncName() != c.name {
				t.Errorf("want .TestFuncName=%s, got=%s", c.name, tpl.TestFuncName())
			}
		})
	}
}

func basicFixture() specs.Fixture {
	return specs.Fixture{
		RootFile:       &specs.FixtureFileInMem{FileBytes: []byte(`{root: '0x44de62c118d7951f5b6d9a03444e54aff47d02ff57add2a4eb2a198b3e83ae35'}`)},
		SerializedFile: &specs.FixtureFileInMem{FileBytes: []byte("serialized-ssz-snappy-bytes")},
		YamlFile:       &specs.FixtureFileInMem{FileBytes: []byte("value: yaml")},
		Directory:      "tests/mainnet/altair/ssz_static/AggregateAndProof/ssz_random/case_0",
	}
}

func TestCaseFileLayout(t *testing.T) {
	fs := afero.NewMemMapFs()
	fix := basicFixture()
	cases := map[specs.TestIdent]specs.Fixture{
		specs.TestIdent{Preset: specs.Mainnet, Fork: specs.Altair, Name: "Checkpoint", Offset: 0}: fix,
	}
	rels := &SpecRelationships{
		Package: "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1",
		Preset:  specs.Mainnet,
		Defs: []ForkTypeDefinitions{
			{
				Fork: specs.Altair,
				Types: []TypeRelation{
					{
						SpecName: "Checkpoint",
					},
				},
			},
		},
	}
	err := WriteSpecTestFiles(cases, rels, fs)
	if err != nil {
		t.Fatalf("failed to write spec test files with error=%s", err.Error())
	}
	// WriteSpecTestFiles writes fixtures under a "testdata/" prefix (see
	// TestCaseTpl.FixtureDirectory). View the fs rooted at "testdata" so spec
	// paths can be referenced naively, the same way the generated test package
	// resolves them relative to its own directory.
	tdfs := afero.NewBasePathFs(fs, "testdata")
	entries, err := afero.ReadDir(tdfs, fix.Directory)
	if err != nil {
		t.Fatalf("failed to list directory %s with error=%s", fix.Directory, err.Error())
	}
	searching := map[string]bool{
		specs.RootFilename:       true,
		specs.SerializedFilename: true,
		specs.ValueFilename:      true,
	}
	for _, f := range entries {
		_, n, err := specs.ParsePath(path.Join(fix.Directory, f.Name()))
		if err != nil {
			t.Fatalf("failed to parse path %s", path.Join(fix.Directory, f.Name()))
		}
		_, ok := searching[n]
		if ok {
			delete(searching, n)
		}
	}
	if len(searching) != 0 {
		for k := range searching {
			t.Errorf("did not find %s in directory entries", k)
		}
	}
}

func TestRenderTestCaseTpl(t *testing.T) {
	tpl := TestCaseTpl{
		ident: specs.TestIdent{
			Preset: specs.Mainnet,
			Fork:   specs.Altair,
			Offset: 0,
		},
		fixture:    basicFixture(),
		structName: "AggregateAttestationAndProof",
	}
	got, err := tpl.Render()
	if err != nil {
		t.Fatalf("error rendering template = %s", err.Error())
	}
	want := `func Test_mainnet_altair_AggregateAttestationAndProof_0(t *testing.T) {
	fixtureDir := "testdata/tests/mainnet/altair/ssz_static/AggregateAndProof/ssz_random/case_0"
	root, serialized, err := specs.RootAndSerializedFromFixture(fixtureDir)
	if err != nil {
		t.Fatalf("error reading fixtures in dir %s", fixtureDir)
	}
	v := &AggregateAttestationAndProof{}
	err = v.UnmarshalSSZ(serialized)
	if err != nil {
		t.Fatalf("error in UnmarshalSSZ reading fixture data from %s, err=%s", fixtureDir, err.Error())
	}
	sroot, err := v.HashTreeRoot()
	if err != nil {
		t.Fatalf("error from HashTreeRoot=%s, from fixture data in %s", err.Error(), fixtureDir)
	}
	if root != sroot {
		t.Fatalf("HashTreeRoot of fixture wrong, want=%#x, got=%#x, from fixture data in %s", root, sroot, fixtureDir)
	}
	encoded, err := v.MarshalSSZ()
	if err != nil {
		t.Fatalf("error in MarshalSSZ for fixture data from %s, err=%s", fixtureDir, err.Error())
	}
	if !bytes.Equal(encoded, serialized) {
		t.Fatalf("MarshalSSZ round trip mismatch for fixture data in %s", fixtureDir)
	}
}`
	if got != want {
		t.Fatalf("rendered template wrong, want=%s, got=%s", want, got)
	}
}
