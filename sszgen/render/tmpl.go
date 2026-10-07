package render

import (
	"bytes"
	"text/template"
)

// execTmpl renders a pre-parsed fragment template. Fragment templates are
// package-level template.Must vars so syntax errors surface at init, not at
// generation time; an execution error here is a programmer error (a data
// struct out of sync with its template), so it panics.
func execTmpl(t *template.Template, data any) string {
	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// fieldElements parameterizes the fragments whose only input is the field
// reference being read or written.
type fieldElements struct {
	FieldName string
}

// nilInitTmpl guards a field whose zero value cannot be used directly: allocate
// it before sizing or marshaling. Shared by the size and marshal assemblers.
var nilInitTmpl = template.Must(template.New("nilInit").Parse(
	`if {{.FieldName}} == nil {
	{{.FieldName}} = {{.Init}}
}`))

type nilInitElements struct {
	FieldName string
	Init      string
}
