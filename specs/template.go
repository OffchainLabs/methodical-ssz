package specs

import "text/template"

var testCaseTemplateBytes = `func {{.TestFuncName}}(t *testing.T) {
	fixtureDir := "{{.FixtureDirectory}}"
	root, serialized, err := specs.RootAndSerializedFromFixture(fixtureDir)
	require.NoError(t, err)
	v := &{{.GoTypeName}}{}
	require.NoError(t, v.UnmarshalSSZ(serialized))
	sroot, err := v.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, root, sroot)
}`

var testCaseTemplateImports = `import (
	"testing"

	"github.com/OffchainLabs/methodical-ssz/specs"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)`

var testFuncBodyTpl *template.Template

func init() {
	testFuncBodyTpl = template.Must(template.New("testFuncBodyTpl").Parse(testCaseTemplateBytes))
}
