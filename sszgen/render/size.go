package render

import (
	"bytes"
	"fmt"
	"go/types"
	"strings"
	"text/template"

	"github.com/OffchainLabs/methodical-ssz/sszgen/core"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// This file is the complete SizeSSZ operation: how every SSZ kind contributes to
// a type's encoded size, in one place.
//
// sizeOp is parameterized by its accumulator variable so it can be reused with a
// different target: SizeSSZ accumulates into "size", while marshal's offset
// bookkeeping reuses the same per-kind logic accumulating into "offset".

const (
	sizeReceiver = "c"
	sizeVar      = "size"
)

// SizeFragment is one field's contribution to a SizeSSZ body. The fixed byte
// count is summed by the assembler straight from ValRep.FixedSize(), so the op
// produces only the runtime pieces: an optional nil-init expression (for the
// `if f == nil { f = <Init> }` guard) and the variable-size statement.
type SizeFragment struct {
	Init     string
	Variable string
}

// SizeMethodSet exposes this operation to the render orchestration.
func SizeMethodSet() core.MethodSet {
	return core.MethodSet{Name: "size", Generate: generateSize}
}

var sizeBodyTmpl = template.Must(template.New("sizeBody").Parse(
	`func ({{.Receiver}} {{.Type}}) SizeSSZ() (int) {
	size := {{.FixedSize}}
	{{- .VariableSize}}
	return size
}`))

func generateSize(vc *gentypes.ValueContainer, ctx *core.GenContext) ([]string, error) {
	op := sizeOp{acc: sizeVar}
	fixed := 0
	comps := make([]string, 0)
	for _, field := range vc.Contents {
		fixed += field.Value.FixedSize()
		if !field.Value.IsVariableSized() {
			continue
		}
		ref := fmt.Sprintf("%s.%s", sizeReceiver, field.Key)
		frag := core.Dispatch(op, field.Value, ref, ctx)
		if frag.Init != "" {
			comps = append(comps, execTmpl(nilInitTmpl, nilInitElements{FieldName: ref, Init: frag.Init}))
		}
		if frag.Variable != "" {
			comps = append(comps, "\t"+frag.Variable)
		}
	}
	buf := bytes.NewBuffer(nil)
	err := sizeBodyTmpl.Execute(buf, struct {
		Receiver     string
		Type         string
		FixedSize    int
		VariableSize string
	}{
		Receiver:     sizeReceiver,
		Type:         "*" + vc.TypeName(),
		FixedSize:    fixed,
		VariableSize: "\n" + strings.Join(comps, "\n"),
	})
	if err != nil {
		return nil, err
	}
	return []string{buf.String()}, nil
}

// sizeOp is the per-kind visitor for the size operation. acc is the accumulator
// variable the per-kind statements add to ("size", or "offset" when reused by
// marshal's offset bookkeeping).
type sizeOp struct{ acc string }

func (sizeOp) Name() string                                    { return "size" }
func (sizeOp) Delegates(ctx *core.GenContext) *types.Interface { return ctx.Ifaces().Sizer }

// Delegate: the field type already implements SizeSSZ; add its runtime size.
func (o sizeOp) Delegate(vr gentypes.ValRep, ref string, ctx *core.GenContext) SizeFragment {
	frag := SizeFragment{Variable: fmt.Sprintf("%s += %s.SizeSSZ()", o.acc, ref)}
	if ptr, ok := vr.(*gentypes.ValuePointer); ok {
		frag.Init = fmt.Sprintf("new(%s)", ctx.QualifiedTypeName(ptr.Referent))
	}
	return frag
}

// Fixed scalars contribute nothing at runtime.
func (sizeOp) Bool(*gentypes.ValueBool, string, *core.GenContext) SizeFragment { return SizeFragment{} }
func (sizeOp) Byte(*gentypes.ValueByte, string, *core.GenContext) SizeFragment { return SizeFragment{} }
func (sizeOp) Uint(*gentypes.ValueUint, string, *core.GenContext) SizeFragment { return SizeFragment{} }
func (sizeOp) Union(*gentypes.ValueUnion, string, *core.GenContext) SizeFragment {
	panic("union types are not supported")
}

func (o sizeOp) List(v *gentypes.ValueList, ref string, ctx *core.GenContext) SizeFragment {
	if !v.ElementValue.IsVariableSized() {
		if v.ElementValue.FixedSize() > 1 {
			return SizeFragment{Variable: fmt.Sprintf("%s += len(%s) * %d", o.acc, ref, v.ElementValue.FixedSize())}
		}
		return SizeFragment{Variable: fmt.Sprintf("%s += len(%s)", o.acc, ref)}
	}
	elem := core.Dispatch(o, v.ElementValue, "o", ctx)
	return SizeFragment{Variable: execTmpl(variableSizedListTmpl, sizeLoopElements{
		FieldName:       ref,
		VarName:         o.acc,
		SizeComputation: elem.Variable,
	})}
}

func (o sizeOp) Vector(v *gentypes.ValueVector, ref string, ctx *core.GenContext) SizeFragment {
	// Only reached for a vector with variable-sized elements (a fixed vector is
	// never dispatched here). Its encoding is the same as a variable list's —
	// a 4-byte offset plus the element's encoding, per element — so the size
	// loop is the list's.
	elem := core.Dispatch(o, v.ElementValue, "o", ctx)
	return SizeFragment{Variable: execTmpl(variableSizedListTmpl, sizeLoopElements{
		FieldName:       ref,
		VarName:         o.acc,
		SizeComputation: elem.Variable,
	})}
}

func (o sizeOp) Container(v *gentypes.ValueContainer, ref string, ctx *core.GenContext) SizeFragment {
	return SizeFragment{
		Init:     fmt.Sprintf("new(%s)", ctx.QualifiedTypeName(v)),
		Variable: fmt.Sprintf("%s += %s.SizeSSZ()", o.acc, ref),
	}
}

func (o sizeOp) Overlay(v *gentypes.ValueOverlay, ref string, ctx *core.GenContext) SizeFragment {
	if !v.Underlying.IsVariableSized() {
		return SizeFragment{}
	}
	return core.Dispatch(o, v.Underlying, ref, ctx)
}

func (o sizeOp) Pointer(v *gentypes.ValuePointer, ref string, ctx *core.GenContext) SizeFragment {
	return core.Dispatch(o, v.Referent, ref, ctx)
}

var variableSizedListTmpl = template.Must(template.New("variableSizedList").Parse(
	`for _, o := range {{.FieldName}} {
		{{.VarName}} += 4
		{{.SizeComputation}}
	}`))

// sizeLoopElements parameterizes the per-element size loops: the field being
// ranged over, the accumulator variable, and the element's size statement.
type sizeLoopElements struct {
	FieldName       string
	VarName         string
	SizeComputation string
}
