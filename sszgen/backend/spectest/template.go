package spectest

import "text/template"

var testCaseTemplateBytes = `func {{.TestFuncName}}(t *testing.T) {
	fixtureDir := "{{.FixtureDirectory}}"
	root, serialized, err := specs.RootAndSerializedFromFixture(fixtureDir)
	if err != nil {
		t.Fatalf("error reading fixtures in dir %s", fixtureDir)
	}
	v := &{{.GoTypeName}}{}
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

var testCaseTemplateImports = `import (
	"bytes"
	"testing"

	"github.com/OffchainLabs/methodical-ssz/specs"
)`

var testFuncBodyTpl *template.Template

func init() {
	testFuncBodyTpl = template.Must(template.New("testFuncBodyTpl").Parse(testCaseTemplateBytes))
}
