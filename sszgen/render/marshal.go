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

// This file is the complete MarshalSSZ/MarshalSSZTo operation: how every SSZ kind
// writes its fixed-section bytes (or offset) and its variable-section bytes, in
// one place.

const marshalReceiver = "c"

// MarshalFragment is one field's contribution to MarshalSSZTo, split by the two
// sections the method emits: Fixed (the fixed-section write, or an offset write
// for a variable field) and Variable (the variable-section write, "" for fixed
// fields). The nil-init guard is computed by the assembler via initValue.
type MarshalFragment struct {
	Fixed    string
	Variable string
}

// MarshalMethodSet exposes this operation to the render orchestration.
func MarshalMethodSet() core.MethodSet {
	return core.MethodSet{Name: "marshal", Generate: generateMarshal}
}

var marshalBodyTmpl = template.Must(template.New("marshalBody").Parse(
	`func ({{.Receiver}} {{.Type}}) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, {{.Receiver}}.SizeSSZ())
	return {{.Receiver}}.MarshalSSZTo(buf[:0])
}

func ({{.Receiver}} {{.Type}}) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
{{- .OffsetDeclaration -}}
{{- .ValueMarshaling }}
{{- .VariableValueMarshaling }}
	return dst, err
}`))

func generateMarshal(vc *gentypes.ValueContainer, ctx *core.GenContext) ([]string, error) {
	fixedBlocks := make([]string, 0)
	varBlocks := make([]string, 0)
	offset := 0
	for i, c := range vc.Contents {
		fieldName := fmt.Sprintf("%s.%s", marshalReceiver, c.Key)
		frag := core.Dispatch(marshalOp{field: c.Key}, c.Value, fieldName, ctx)

		fixedBlocks = append(fixedBlocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
		if ini := initValue(c.Value, ctx, ctx.Ifaces().Marshaler); ini != "" {
			fixedBlocks = append(fixedBlocks, execTmpl(nilInitTmpl, nilInitElements{FieldName: fieldName, Init: ini}))
		}
		fixedBlocks = append(fixedBlocks, "\t"+frag.Fixed)
		offset += c.Value.FixedSize()

		if c.Value.IsVariableSized() && frag.Variable != "" {
			varBlocks = append(varBlocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
			varBlocks = append(varBlocks, "\t"+frag.Variable)
		}
	}

	offsetDecl := ""
	if vc.IsVariableSized() {
		offsetDecl = fmt.Sprintf("\noffset := %d\n", offset)
	}
	buf := bytes.NewBuffer(nil)
	err := marshalBodyTmpl.Execute(buf, struct {
		Receiver                string
		Type                    string
		OffsetDeclaration       string
		ValueMarshaling         string
		VariableValueMarshaling string
	}{
		Receiver:                marshalReceiver,
		Type:                    "*" + vc.TypeName(),
		OffsetDeclaration:       offsetDecl,
		ValueMarshaling:         "\n" + strings.Join(fixedBlocks, "\n"),
		VariableValueMarshaling: "\n" + strings.Join(varBlocks, "\n"),
	})
	if err != nil {
		return nil, err
	}
	return []string{buf.String()}, nil
}

func delegateMarshal(ref, field string) string {
	return execTmpl(delegateMarshalTmpl, delegateMarshalElements{FieldName: ref, Field: field})
}

// offsetWrite is the fixed-section write for a variable field: emit a 4-byte
// offset, then advance the running offset by the field's size.
func offsetWrite(ref string) string {
	return execTmpl(offsetWriteTmpl, fieldElements{FieldName: ref})
}

// marshalOp is the per-kind visitor for the marshal operation. field carries the
// clean name of the enclosing struct field (no receiver prefix), threaded down
// through pointers, overlays, and collection elements so a nested MarshalSSZTo
// failure can wrap its error with the field where it occurred.
type marshalOp struct{ field string }

func (marshalOp) Name() string                                    { return "marshal" }
func (marshalOp) Delegates(ctx *core.GenContext) *types.Interface { return ctx.Ifaces().Marshaler }

func (o marshalOp) Delegate(vr gentypes.ValRep, ref string, _ *core.GenContext) MarshalFragment {
	if vr.IsVariableSized() {
		return MarshalFragment{Fixed: offsetWrite(ref), Variable: delegateMarshal(ref, o.field)}
	}
	return MarshalFragment{Fixed: delegateMarshal(ref, o.field)}
}

func (marshalOp) Byte(_ *gentypes.ValueByte, ref string, _ *core.GenContext) MarshalFragment {
	return MarshalFragment{Fixed: fmt.Sprintf("dst = append(dst, %s)", ref)}
}
func (marshalOp) Union(*gentypes.ValueUnion, string, *core.GenContext) MarshalFragment {
	panic("union types are not supported")
}

func (marshalOp) Uint(v *gentypes.ValueUint, ref string, _ *core.GenContext) MarshalFragment {
	if v.Size > 64 {
		// wide uints are little-endian limb arrays ([4]uint64 for uint256);
		// serializing the limbs in order is the spec's little-endian encoding.
		// The limb count is known at generation time, so emit unrolled lines.
		lines := make([]string, int(v.Size)/64)
		for i := range lines {
			lines[i] = fmt.Sprintf("dst = binary.LittleEndian.AppendUint64(dst, %s[%d])", ref, i)
		}
		return MarshalFragment{Fixed: strings.Join(lines, "\n")}
	}
	return MarshalFragment{Fixed: fmt.Sprintf("dst = binary.LittleEndian.AppendUint%d(dst, %s)", v.Size, ref)}
}

func (marshalOp) Bool(_ *gentypes.ValueBool, ref string, _ *core.GenContext) MarshalFragment {
	return MarshalFragment{Fixed: execTmpl(marshalBoolTmpl, fieldElements{FieldName: ref})}
}

func (o marshalOp) Container(v *gentypes.ValueContainer, ref string, _ *core.GenContext) MarshalFragment {
	if v.IsVariableSized() {
		return MarshalFragment{Fixed: offsetWrite(ref), Variable: delegateMarshal(ref, o.field)}
	}
	return MarshalFragment{Fixed: delegateMarshal(ref, o.field)}
}

func (o marshalOp) Pointer(v *gentypes.ValuePointer, ref string, ctx *core.GenContext) MarshalFragment {
	return core.Dispatch(marshalOp{field: o.field}, v.Referent, ref, ctx)
}

func (o marshalOp) Overlay(v *gentypes.ValueOverlay, ref string, ctx *core.GenContext) MarshalFragment {
	// The fixed write coerces the ref to the underlying type; the variable write
	// (only for variable underlyings) uses the plain ref.
	frag := MarshalFragment{Fixed: core.Dispatch(marshalOp{field: o.field}, v.Underlying, core.Coerce(v.Underlying)(ref), ctx).Fixed}
	if v.Underlying.IsVariableSized() {
		frag.Variable = core.Dispatch(marshalOp{field: o.field}, v.Underlying, ref, ctx).Variable
	}
	return frag
}

func (o marshalOp) Vector(v *gentypes.ValueVector, ref string, ctx *core.GenContext) MarshalFragment {
	if v.IsVariableSized() {
		// A vector of variable-sized elements serializes like a variable list —
		// per-element offsets then element payloads — but with an exact length
		// check instead of a max-size check. Fixed section: write the field's
		// offset, advance it by the vector's encoded size.
		vecSize := core.Dispatch(sizeOp{acc: "offset"}, v, ref, ctx).Variable
		fixed := execTmpl(marshalListOffsetTmpl, marshalListOffsetElements{SizeComputation: vecSize})
		return MarshalFragment{Fixed: fixed, Variable: marshalVectorVariable(v, ref, ctx, o.field)}
	}
	var marshalValue string
	if _, isByte := v.ElementValue.(*gentypes.ValueByte); isByte {
		if v.IsArray {
			marshalValue = fmt.Sprintf("dst = append(dst, %s[:]...)", ref)
		} else {
			marshalValue = fmt.Sprintf("dst = append(dst, %s...)", ref)
		}
	} else {
		nested := nestedMarshalName(ref)
		internal := core.Dispatch(marshalOp{field: o.field}, v.ElementValue, nested, ctx).Fixed
		marshalValue = execTmpl(rangeLoopTmpl, rangeLoopElements{NestedFieldName: nested, FieldName: ref, Body: internal})
	}
	fixed := execTmpl(tmplGenerateMarshalValueVector, marshalVectorElements{
		FieldName:    ref,
		Size:         v.Size,
		MarshalValue: marshalValue,
	})
	return MarshalFragment{Fixed: fixed}
}

func (o marshalOp) List(v *gentypes.ValueList, ref string, ctx *core.GenContext) MarshalFragment {
	// Fixed: write the offset, then advance it by the list's runtime size.
	listSize := core.Dispatch(sizeOp{acc: "offset"}, v, ref, ctx).Variable
	fixed := execTmpl(marshalListOffsetTmpl, marshalListOffsetElements{SizeComputation: listSize})
	return MarshalFragment{Fixed: fixed, Variable: marshalListVariable(v, ref, ctx, o.field)}
}

// marshalVectorVariable writes a variable vector's payload: per-element offsets,
// then each element's variable encoding. field is the enclosing struct field,
// so a nested element's MarshalSSZTo error wraps with the field where it failed.
func marshalVectorVariable(v *gentypes.ValueVector, ref string, ctx *core.GenContext, field string) string {
	nested := nestedMarshalName(ref)
	internal := core.Dispatch(marshalOp{field: field}, v.ElementValue, nested, ctx).Variable
	offsetMgmt := execTmpl(tmplVariableOffsetManagement, offsetManagementElements{
		FieldName:       ref,
		NestedFieldName: nested,
		SizeComputation: core.Dispatch(sizeOp{acc: "offset"}, v.ElementValue, nested, ctx).Variable,
	})
	marshalValue := execTmpl(rangeLoopTmpl, rangeLoopElements{NestedFieldName: nested, FieldName: ref, Body: internal})
	return execTmpl(tmplGenerateMarshalValueVector, marshalVectorElements{
		FieldName:        ref,
		Size:             v.Size,
		OffsetManagement: offsetMgmt,
		MarshalValue:     marshalValue,
	})
}

func marshalListVariable(v *gentypes.ValueList, ref string, ctx *core.GenContext, field string) string {
	elem := v.ElementValue
	var marshalValue, offsetMgmt string
	if _, isByte := elem.(*gentypes.ValueByte); isByte {
		marshalValue = fmt.Sprintf("dst = append(dst, %s...)", ref)
	} else {
		nested := nestedMarshalName(ref)
		var internal string
		if elem.IsVariableSized() {
			internal = core.Dispatch(marshalOp{field: field}, elem, nested, ctx).Variable
			offsetMgmt = execTmpl(tmplVariableOffsetManagement, offsetManagementElements{
				FieldName:       ref,
				NestedFieldName: nested,
				SizeComputation: core.Dispatch(sizeOp{acc: "offset"}, elem, nested, ctx).Variable,
			})
		} else {
			internal = core.Dispatch(marshalOp{field: field}, elem, nested, ctx).Fixed
		}
		marshalValue = execTmpl(rangeLoopTmpl, rangeLoopElements{NestedFieldName: nested, FieldName: ref, Body: internal})
	}
	if v.Progressive {
		// progressive lists have no limit, hence no max check
		return execTmpl(tmplGenerateMarshalValueProgressiveList, marshalListElements{
			FieldName:        ref,
			MarshalValue:     marshalValue,
			OffsetManagement: offsetMgmt,
		})
	}
	return execTmpl(tmplGenerateMarshalValueList, marshalListElements{
		FieldName:        ref,
		MaxSize:          v.MaxSize,
		MarshalValue:     marshalValue,
		OffsetManagement: offsetMgmt,
	})
}

// marshalVectorElements parameterizes a vector's element writes guarded by the
// exact length check. OffsetManagement is empty for fixed-size elements; for
// variable-sized elements it is the per-element offset loop.
type marshalVectorElements struct {
	FieldName        string
	Size             int
	OffsetManagement string
	MarshalValue     string
}

// marshalListElements parameterizes the variable-section write of a list: the
// max-size check, optional per-element offset management, and element writes.
type marshalListElements struct {
	FieldName        string
	MaxSize          int
	MarshalValue     string
	OffsetManagement string
}

// offsetManagementElements parameterizes the offset bookkeeping loop emitted
// before a list of variable-sized elements is written.
type offsetManagementElements struct {
	FieldName       string
	NestedFieldName string
	SizeComputation string
}

func nestedMarshalName(fieldName string) string {
	if fieldName[0:1] == "o" && monoCharacter(fieldName) {
		return fieldName + "o"
	}
	return "o"
}

// rangeLoopElements parameterizes a plain per-element loop: the loop variable,
// the field ranged over, and the per-element statement(s).
type rangeLoopElements struct {
	NestedFieldName string
	FieldName       string
	Body            string
}

// marshalListOffsetElements parameterizes a list's fixed-section write: the
// statement advancing the running offset by the list's encoded size.
type marshalListOffsetElements struct {
	SizeComputation string
}

// delegateMarshalElements parameterizes the nested MarshalSSZTo call: the field
// reference to marshal and the clean field name its error is wrapped with.
type delegateMarshalElements struct {
	FieldName string
	Field     string
}

var (
	delegateMarshalTmpl = template.Must(template.New("delegateMarshal").Parse(
		`if dst, err = {{.FieldName}}.MarshalSSZTo(dst); err != nil {
	return nil, fmt.Errorf("{{.Field}}: %w", err)
}`))

	offsetWriteTmpl = template.Must(template.New("offsetWrite").Parse(
		`dst = ssz.WriteOffset(dst, offset)
offset += {{.FieldName}}.SizeSSZ()`))

	marshalBoolTmpl = template.Must(template.New("marshalBool").Parse(
		`if {{.FieldName}} {
	dst = append(dst, 1)
} else {
	dst = append(dst, 0)
}`))

	rangeLoopTmpl = template.Must(template.New("rangeLoop").Parse(
		`for _, {{.NestedFieldName}} := range {{.FieldName}} {
	{{.Body}}
}`))

	marshalListOffsetTmpl = template.Must(template.New("marshalListOffset").Parse(
		`dst = ssz.WriteOffset(dst, offset)
{{.SizeComputation}}
`))

	tmplGenerateMarshalValueVector = template.Must(template.New("marshalValueVector").Parse(
		`if len({{.FieldName}}) != {{.Size}} {
	return nil, ssz.ErrBytesLength
}
{{.OffsetManagement}}{{.MarshalValue}}`))

	tmplGenerateMarshalValueList = template.Must(template.New("marshalValueList").Parse(
		`if len({{.FieldName}}) > {{.MaxSize}} {
	return nil, ssz.ErrListTooBig
}
{{.OffsetManagement}}{{.MarshalValue}}`))

	tmplGenerateMarshalValueProgressiveList = template.Must(template.New("marshalValueProgressiveList").Parse(
		`{{.OffsetManagement}}{{.MarshalValue}}`))

	tmplVariableOffsetManagement = template.Must(template.New("variableOffsetManagement").Parse(
		`{
	offset = 4 * len({{.FieldName}})
	for _, {{.NestedFieldName}} := range {{.FieldName}} {
		dst = ssz.WriteOffset(dst, offset)
		{{.SizeComputation}}
	}
}
`))
)
