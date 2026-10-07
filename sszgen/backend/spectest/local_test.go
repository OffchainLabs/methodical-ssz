package spectest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/specs"
	"github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture"
	"github.com/OffchainLabs/methodical-ssz/sszgen/testutil/widgetfixture"
	"github.com/golang/snappy"
	"github.com/spf13/afero"
)

func TestRenderLocalTestCaseTpl(t *testing.T) {
	tpl := TestCaseTpl{
		ident: specs.TestIdent{
			Preset: specs.Mainnet,
			Fork:   specs.Altair,
			Offset: 0,
		},
		fixture:    basicFixture(),
		structName: "AggregateAttestationAndProof",
		qualifier:  "v1alpha1",
	}
	got, err := tpl.Render()
	if err != nil {
		t.Fatalf("error rendering template = %s", err.Error())
	}
	if !strings.Contains(got, "v := &v1alpha1.AggregateAttestationAndProof{}") {
		t.Fatalf("expected qualified type reference, got:\n%s", got)
	}
	if !strings.Contains(got, "func Test_mainnet_altair_AggregateAttestationAndProof_0(t *testing.T)") {
		t.Fatalf("unexpected test func name:\n%s", got)
	}
}

func TestWriteLocalSpecTestFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	fix := basicFixture()
	cases := map[specs.TestIdent]specs.Fixture{
		{Preset: specs.Mainnet, Fork: specs.Altair, Name: "Checkpoint", Offset: 0}: fix,
	}
	rels := &SpecRelationships{
		Package: "github.com/prysmaticlabs/prysm/v7/proto/prysm/v1alpha1",
		Preset:  specs.Mainnet,
		Defs: []ForkTypeDefinitions{
			{Fork: specs.Altair, Types: []TypeRelation{{SpecName: "Checkpoint"}}},
		},
	}
	if err := WriteLocalSpecTestFiles(cases, rels, fs, ""); err != nil {
		t.Fatal(err)
	}
	tb, err := afero.ReadFile(fs, "methodical_test.go")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(tb)
	for _, frag := range []string{
		"package spectest",
		`v1alpha1 "github.com/prysmaticlabs/prysm/v7/proto/prysm/v1alpha1"`,
		"v := &v1alpha1.Checkpoint{}",
		`fixtureDir := "testdata/tests/mainnet/altair/ssz_static/AggregateAndProof/ssz_random/case_0"`,
	} {
		if !strings.Contains(contents, frag) {
			t.Fatalf("missing expected fragment %q in generated test file:\n%s", frag, contents)
		}
	}
	// fixtures land under testdata/ in the output
	exists, err := afero.Exists(fs, "testdata/"+fix.Directory+"/"+specs.RootFilename)
	if err != nil || !exists {
		t.Fatalf("expected fixture root file under testdata (exists=%v, err=%v)", exists, err)
	}
}

// TestGenSpectestEndToEnd exercises the full local mode against real, compiled
// methodsets: it builds a synthetic fixture for delegatefixture.Wide, writes a
// local-mode spec test package into the module's gitignored generated/
// directory, and runs it with `go test` — actually executing generated SSZ
// methods against the harness plumbing.
func TestGenSpectestEndToEnd(t *testing.T) {
	w := &delegatefixture.Wide{1, 2, 3, 4, 5, 6, 7, 8}
	serialized, err := w.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	root, err := w.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	fix := specs.Fixture{
		Directory:      "tests/mainnet/phase0/ssz_static/Wide/ssz_random/case_0",
		RootFile:       specs.NewFixtureFileInMem([]byte(fmt.Sprintf("{root: '%#x'}", root)), 0644),
		SerializedFile: specs.NewFixtureFileInMem(snappy.Encode(nil, serialized), 0644),
		YamlFile:       specs.NewFixtureFileInMem([]byte("value: synthetic"), 0644),
	}
	// a second type from a different in-module package exercises the
	// multi-package import path end to end
	g := &widgetfixture.Gadget{9, 8, 7, 6}
	gSerialized, err := g.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	gRoot, err := g.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	gfix := specs.Fixture{
		Directory:      "tests/mainnet/phase0/ssz_static/Gadget/ssz_random/case_0",
		RootFile:       specs.NewFixtureFileInMem([]byte(fmt.Sprintf("{root: '%#x'}", gRoot)), 0644),
		SerializedFile: specs.NewFixtureFileInMem(snappy.Encode(nil, gSerialized), 0644),
		YamlFile:       specs.NewFixtureFileInMem([]byte("value: synthetic"), 0644),
	}
	cases := map[specs.TestIdent]specs.Fixture{
		{Preset: specs.Mainnet, Fork: specs.Phase0, Name: "Wide", Offset: 0}:   fix,
		{Preset: specs.Mainnet, Fork: specs.Phase0, Name: "Gadget", Offset: 0}: gfix,
	}
	rels := &SpecRelationships{
		Package: "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/delegatefixture",
		Preset:  specs.Mainnet,
		Defs: []ForkTypeDefinitions{
			{Fork: specs.Phase0, Types: []TypeRelation{
				{SpecName: "Wide"},
				{SpecName: "Gadget", Package: "github.com/OffchainLabs/methodical-ssz/sszgen/testutil/widgetfixture"},
			}},
		},
	}

	// generated/ is gitignored; the output must live inside the module so the
	// generated package resolves its imports
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(repoRoot, "generated", "genspectest_e2e")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)
	fs := afero.NewBasePathFs(afero.NewOsFs(), outDir)

	if err := WriteLocalSpecTestFiles(cases, rels, fs, ""); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "./generated/genspectest_e2e/")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated spec tests failed: %v\n%s", err, out)
	}
}

// Compatibility gate: a config using only the default package must produce
// byte-identical output to the single-package implementation (captured in
// testdata/local_default.golden before multi-package support landed).
func TestWriteLocalSpecTestFilesDefaultPackageGolden(t *testing.T) {
	fs := afero.NewMemMapFs()
	cases := map[specs.TestIdent]specs.Fixture{
		{Preset: specs.Mainnet, Fork: specs.Altair, Name: "Checkpoint", Offset: 0}: basicFixture(),
	}
	rels := &SpecRelationships{
		Package: "github.com/prysmaticlabs/prysm/v7/proto/prysm/v1alpha1",
		Preset:  specs.Mainnet,
		Defs:    []ForkTypeDefinitions{{Fork: specs.Altair, Types: []TypeRelation{{SpecName: "Checkpoint"}}}},
	}
	if err := WriteLocalSpecTestFiles(cases, rels, fs, ""); err != nil {
		t.Fatal(err)
	}
	got, err := afero.ReadFile(fs, "methodical_test.go")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/local_default.golden")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("default-package output changed:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func fixtureAt(dir string) specs.Fixture {
	f := basicFixture()
	f.Directory = dir
	return f
}

// Multi-package config: types resolve to their own packages, each import is
// aliased deterministically (same-last-element collisions disambiguated), and
// only packages with matched fixtures are imported.
func TestWriteLocalSpecTestFilesMultiPackage(t *testing.T) {
	fs := afero.NewMemMapFs()
	cases := map[specs.TestIdent]specs.Fixture{
		{Preset: specs.Mainnet, Fork: specs.Altair, Name: "Checkpoint", Offset: 0}:          fixtureAt("tests/mainnet/altair/ssz_static/Checkpoint/ssz_random/case_0"),
		{Preset: specs.Mainnet, Fork: specs.Bellatrix, Name: "ExecutionPayload", Offset: 0}: fixtureAt("tests/mainnet/bellatrix/ssz_static/ExecutionPayload/ssz_random/case_0"),
		{Preset: specs.Mainnet, Fork: specs.Bellatrix, Name: "Widget", Offset: 0}:           fixtureAt("tests/mainnet/bellatrix/ssz_static/Widget/ssz_random/case_0"),
	}
	rels := &SpecRelationships{
		Package: "github.com/example/proto/prysm/v1alpha1",
		Preset:  specs.Mainnet,
		Defs: []ForkTypeDefinitions{
			{Fork: specs.Altair, Types: []TypeRelation{
				{SpecName: "Checkpoint"}, // default package
			}},
			{Fork: specs.Bellatrix, Types: []TypeRelation{
				{SpecName: "ExecutionPayload", Package: "github.com/example/proto/engine/v1"},
				// same last path element as engine/v1: forces alias disambiguation
				{SpecName: "Widget", TypeName: "WidgetV1", Package: "github.com/example/proto/widget/v1"},
			}},
		},
	}
	if err := WriteLocalSpecTestFiles(cases, rels, fs, ""); err != nil {
		t.Fatal(err)
	}
	tb, err := afero.ReadFile(fs, "methodical_test.go")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(tb)
	for _, frag := range []string{
		// sorted import order: engine/v1 < prysm/v1alpha1 < widget/v1;
		// engine/v1 claims the bare "v1" alias, widget/v1 disambiguates
		"\tv1 \"github.com/example/proto/engine/v1\"\n\tv1alpha1 \"github.com/example/proto/prysm/v1alpha1\"\n\twidgetv1 \"github.com/example/proto/widget/v1\"",
		"v := &v1alpha1.Checkpoint{}",
		"v := &v1.ExecutionPayload{}",
		"v := &widgetv1.WidgetV1{}",
	} {
		if !strings.Contains(contents, frag) {
			t.Fatalf("missing expected fragment %q in generated test file:\n%s", frag, contents)
		}
	}
}

// A configured override package whose types never match a fixture must not be
// imported — and that includes the default package.
func TestWriteLocalSpecTestFilesUnusedPackagesOmitted(t *testing.T) {
	fs := afero.NewMemMapFs()
	cases := map[specs.TestIdent]specs.Fixture{
		{Preset: specs.Mainnet, Fork: specs.Bellatrix, Name: "ExecutionPayload", Offset: 0}: fixtureAt("tests/mainnet/bellatrix/ssz_static/ExecutionPayload/ssz_random/case_0"),
	}
	rels := &SpecRelationships{
		Package: "github.com/example/proto/prysm/v1alpha1",
		Preset:  specs.Mainnet,
		Defs: []ForkTypeDefinitions{
			{Fork: specs.Bellatrix, Types: []TypeRelation{
				{SpecName: "ExecutionPayload", Package: "github.com/example/proto/engine/v1"},
				// configured in another package, but no fixture matches it
				{SpecName: "Sidecar", Package: "github.com/example/proto/sidecar/v1"},
			}},
		},
	}
	if err := WriteLocalSpecTestFiles(cases, rels, fs, ""); err != nil {
		t.Fatal(err)
	}
	tb, err := afero.ReadFile(fs, "methodical_test.go")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(tb)
	if !strings.Contains(contents, `v1 "github.com/example/proto/engine/v1"`) {
		t.Fatalf("missing engine/v1 import:\n%s", contents)
	}
	for _, absent := range []string{"v1alpha1", "sidecar"} {
		if strings.Contains(contents, absent) {
			t.Fatalf("unexpected unused import reference %q:\n%s", absent, contents)
		}
	}
}

// A later fork can move a spec type to a different package; the resolved
// package follows the most recent definition at or before the target fork.
func TestRelationsAtForkPackageOverride(t *testing.T) {
	sr := SpecRelationships{
		Package: "github.com/example/default",
		Defs: []ForkTypeDefinitions{
			{Fork: specs.Bellatrix, Types: []TypeRelation{
				{SpecName: "ExecutionPayload", Package: "github.com/example/engine/v1"},
			}},
			{Fork: specs.Capella, Types: []TypeRelation{
				{SpecName: "ExecutionPayload", TypeName: "ExecutionPayloadCapella", Package: "github.com/example/engine/v2"},
			}},
		},
	}
	atBellatrix, err := sr.RelationsAtFork(specs.Bellatrix)
	if err != nil {
		t.Fatal(err)
	}
	if rt := atBellatrix["ExecutionPayload"]; rt.Package != "github.com/example/engine/v1" || rt.TypeName != "ExecutionPayload" {
		t.Fatalf("bellatrix resolution wrong: %+v", rt)
	}
	atCapella, err := sr.RelationsAtFork(specs.Capella)
	if err != nil {
		t.Fatal(err)
	}
	if rt := atCapella["ExecutionPayload"]; rt.Package != "github.com/example/engine/v2" || rt.TypeName != "ExecutionPayloadCapella" {
		t.Fatalf("capella resolution wrong: %+v", rt)
	}
}
