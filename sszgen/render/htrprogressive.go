package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// This file is the complete progressive merkleization operation
// (ProgressiveContainer, merkleize_progressive, mix_in_active_fields). Types
// marked progressive in the generator config get ProgressiveHashTreeRoot /
// ProgressiveHashTreeRootWith; their standard HashTreeRoot[With] methods are
// thin wrappers delegating here.

// ProgressiveHashTreeRootMethodSet exposes this operation to the render
// orchestration. It emits nothing for types not marked progressive.
func ProgressiveHashTreeRootMethodSet() core.MethodSet {
	return core.MethodSet{Name: "htr-progressive", Generate: generateProgressiveHTR}
}

var progressiveHTRTmpl = template.Must(template.New("htrProgressive").Parse(
	`var {{.ActiveFieldsVar}} = {{.ActiveFieldsBytes}}

func ({{.Receiver}} {{.Type}}) ProgressiveHashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := {{.Receiver}}.ProgressiveHashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func ({{.Receiver}} {{.Type}}) ProgressiveHashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	{{.HTRSteps}}
	hh.MerkleizeProgressiveWithActiveFields(indx, {{.ActiveFieldsVar}})
	return nil
}`))

func generateProgressiveHTR(vc *gentypes.ValueContainer, ctx *core.GenContext) ([]string, error) {
	if vc.ActiveFields == nil {
		return nil, nil
	}
	steps := make([]string, 0)
	for i, c := range vc.Contents {
		ref := fmt.Sprintf("%s.%s", htrReceiver, c.Key)
		steps = append(steps, fmt.Sprintf("\t// Field %d: %s", i, c.Key))
		steps = append(steps, core.Dispatch(htrOp{field: c.Key}, c.Value, ref, ctx))
	}
	buf := bytes.NewBuffer(nil)
	err := progressiveHTRTmpl.Execute(buf, struct {
		Receiver          string
		Type              string
		HTRSteps          string
		ActiveFieldsVar   string
		ActiveFieldsBytes string
	}{
		Receiver:          htrReceiver,
		Type:              "*" + vc.TypeName(),
		HTRSteps:          strings.Join(steps, "\n"),
		ActiveFieldsVar:   "activeFields" + vc.TypeName(),
		ActiveFieldsBytes: encodeActiveFields(vc.ActiveFields),
	})
	if err != nil {
		return nil, err
	}
	return []string{buf.String()}, nil
}

// encodeActiveFields packs an active-fields bitvector per the spec's pack_bits
// (bit i is bit i%8 of byte i/8) and renders it as a Go byte-slice literal.
func encodeActiveFields(af []bool) string {
	b := make([]byte, (len(af)+7)/8)
	for i, active := range af {
		if active {
			b[i/8] |= 1 << uint(i%8)
		}
	}
	sbin := make([]string, len(b))
	for i, v := range b {
		sbin[i] = fmt.Sprintf("0b%08b", v)
	}
	return fmt.Sprintf("[]byte{%s}", strings.Join(sbin, ", "))
}

// progressiveListPutter hashes a ProgressiveList field: same element appends
// as the regular list putter, but no max-size check (progressive lists are
// unlimited) and a progressive merkleize with the length mixin.
func progressiveListPutter(v *gentypes.ValueList, ref, nested string, coerce core.Rewriter, vr gentypes.ValRep, ctx *core.GenContext, field string) string {
	lpe := listPutterElements{FieldName: ref, NestedFieldName: nested}
	if bl := bitlistElement(v.ElementValue); bl != nil {
		lpe.AppendCall = core.Dispatch(htrOp{field: field}, bl, nested, ctx)
		return execTmpl(progressiveListHTRPutterTmpl, lpe)
	}
	switch ev := vr.(type) {
	case *gentypes.ValueByte:
		return execTmpl(progressiveByteListHTRPutterTmpl, fieldElements{FieldName: ref})
	case *gentypes.ValueUint:
		if ev.Size > 64 {
			lpe.AppendCall = htrWideUint(nested, ev.Size)
		} else {
			lpe.AppendCall = fmt.Sprintf("hh.AppendUint%d(%s)", ev.Size, coerce(nested))
		}
		if ev.FixedSize()%htrChunkSize != 0 {
			lpe.PadCall = "\nhh.FillUpTo32()"
		}
		return execTmpl(progressiveListHTRPutterTmpl, lpe)
	case *gentypes.ValueBool:
		lpe.AppendCall = fmt.Sprintf("hh.AppendBool(%s)", coerce(nested))
		lpe.PadCall = "\nhh.FillUpTo32()"
		return execTmpl(progressiveListHTRPutterTmpl, lpe)
	case *gentypes.ValueVector:
		if isByteVector(ev) {
			lpe.AppendCall = renderByteSliceAppend(ev, nested)
		} else {
			lpe.AppendCall = core.Dispatch(htrOp{field: field}, ev, nested, ctx)
		}
		return execTmpl(progressiveListHTRPutterTmpl, lpe)
	case *gentypes.ValueContainer, *gentypes.ValueList:
		lpe.AppendCall = core.Dispatch(htrOp{field: field}, vr, nested, ctx)
		return execTmpl(progressiveListHTRPutterTmpl, lpe)
	default:
		panic(fmt.Sprintf("unsupported type combination - progressive list of %v", ev))
	}
}

var (
	progressiveListHTRPutterTmpl = template.Must(template.New("progressiveListHTRPutter").Parse(`{
	subIndx := hh.Index()
	for _, {{.NestedFieldName}} := range {{.FieldName}} {
		{{.AppendCall}}
	}
	{{- .PadCall}}
	hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len({{.FieldName}})))
}`))

	progressiveByteListHTRPutterTmpl = template.Must(template.New("progressiveByteListHTRPutter").Parse(`{
	subIndx := hh.Index()
	hh.AppendBytes32({{.FieldName}})
	hh.MerkleizeProgressiveWithMixin(subIndx, uint64(len({{.FieldName}})))
}`))
)
