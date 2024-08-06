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

// This file is the complete UnmarshalSSZ operation: the fixed/variable slice
// machinery plus how every SSZ kind reads its value out of a byte slice, in one
// place.

const unmarshalReceiver = "c"

// unmarshalRef is the threaded reference for the unmarshal op: the destination
// field, the source byte slice, and an output caster (set by an enclosing overlay
// so the produced value is wrapped in the overlay's named type). Cast is the
// identity unless an overlay set it. Depth counts enclosing element loops so
// each nesting level gets unique temp names (tmp, tmp2, ...) — reusing one name
// would shadow the outer temp and generate uncompilable code.
type unmarshalRef struct {
	FieldName string
	SliceName string
	Cast      core.Rewriter
	Depth     int
	// Field is the clean name of the enclosing struct field (no receiver
	// prefix), threaded down so a nested method call or a collection element
	// failure can wrap its error with the field where it occurred. It is
	// preserved across element loops so every level contributes its field name.
	Field string
}

// tmpName / tmpSliceName are the element temp names at a given loop depth (depth
// 1 uses the bare name).
func tmpName(depth int) string {
	if depth <= 1 {
		return "tmp"
	}
	return fmt.Sprintf("tmp%d", depth)
}

func tmpSliceName(depth int) string {
	if depth <= 1 {
		return "tmpSlice"
	}
	return fmt.Sprintf("tmpSlice%d", depth)
}

func identity(s string) string { return s }

// UnmarshalMethodSet exposes this operation to the render orchestration.
func UnmarshalMethodSet() core.MethodSet {
	return core.MethodSet{Name: "unmarshal", Generate: generateUnmarshal}
}

var unmarshalTmpl = template.Must(template.New("unmarshal").Parse(
	`func ({{.Receiver}} {{.Type}}) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size {{.SizeInequality}} {{.FixedOffset}} {
		return ssz.ErrSize
	}

	{{.SliceDeclaration}}
{{.ValueUnmarshaling}}
	return err
}`))

func generateUnmarshal(vc *gentypes.ValueContainer, ctx *core.GenContext) ([]string, error) {
	sizeInequality := "!="
	if vc.IsVariableSized() {
		sizeInequality = "<"
	}
	steps := buildUnmarshalSteps(vc)
	fixedOff := unmarshalFixedOffset(vc)

	blocks := make([]string, 0)
	for i, c := range vc.Contents {
		blocks = append(blocks, fmt.Sprintf("\n\t// Field %d: %s", i, c.Key))
		fieldName := fmt.Sprintf("%s.%s", unmarshalReceiver, c.Key)
		if ini := initValue(c.Value, ctx, ctx.Ifaces().Unmarshaler); ini != "" {
			blocks = append(blocks, fmt.Sprintf("%s = %s", fieldName, ini))
		}
		ref := unmarshalRef{FieldName: fieldName, SliceName: sliceName(i), Cast: identity, Field: c.Key}
		mv := core.Dispatch(unmarshalOp{}, c.Value, ref, ctx)
		if mv != "" {
			blocks = append(blocks, mv)
		}
	}

	sliceDecls := strings.Join([]string{steps.fixedSlices(), "", steps.variableSlices(fixedOff)}, "\n")
	buf := bytes.NewBuffer(nil)
	err := unmarshalTmpl.Execute(buf, struct {
		Receiver          string
		Type              string
		SizeInequality    string
		FixedOffset       int
		SliceDeclaration  string
		ValueUnmarshaling string
	}{
		Receiver:          unmarshalReceiver,
		Type:              "*" + vc.TypeName(),
		SizeInequality:    sizeInequality,
		FixedOffset:       fixedOff,
		SliceDeclaration:  sliceDecls,
		ValueUnmarshaling: strings.Join(blocks, "\n"),
	})
	if err != nil {
		return nil, err
	}
	return []string{buf.String()}, nil
}

// initValue is the allocation expression a kind needs before unmarshaling into it
// (a pointer/container is allocated with new(T)); "" for kinds that need none.
//
// A field that already implements UnmarshalSSZ (delegates) is allocated via the
// delegate path: any delegated *pointer gets new(referent), regardless of the
// referent's kind. Otherwise the static type decides.
func initValue(vr gentypes.ValRep, ctx *core.GenContext, iface *types.Interface) string {
	if iface != nil && vr.SatisfiesInterface(iface) {
		if ptr, ok := vr.(*gentypes.ValuePointer); ok {
			return fmt.Sprintf("new(%s)", ctx.QualifiedTypeName(ptr.Referent))
		}
		return ""
	}
	switch v := vr.(type) {
	case *gentypes.ValueContainer:
		return fmt.Sprintf("new(%s)", ctx.QualifiedTypeName(v))
	case *gentypes.ValuePointer:
		return initValue(v.Referent, ctx, iface)
	}
	return ""
}

// unmarshalOp is the per-kind visitor for the UnmarshalSSZ operation.
type unmarshalOp struct{}

func (unmarshalOp) Name() string                                    { return "unmarshal" }
func (unmarshalOp) Delegates(ctx *core.GenContext) *types.Interface { return ctx.Ifaces().Unmarshaler }

func (unmarshalOp) Delegate(_ gentypes.ValRep, ref unmarshalRef, _ *core.GenContext) string {
	return execTmpl(unmarshalCallTmpl, unmarshalCallElements{FieldName: ref.FieldName, SliceName: ref.SliceName, Field: ref.Field})
}

func (unmarshalOp) Byte(_ *gentypes.ValueByte, ref unmarshalRef, _ *core.GenContext) string {
	return fmt.Sprintf("%s = %s", ref.FieldName, ref.Cast(ref.SliceName+"[0]"))
}

func (unmarshalOp) Union(*gentypes.ValueUnion, unmarshalRef, *core.GenContext) string {
	panic("union types are not supported")
}

func (unmarshalOp) Uint(v *gentypes.ValueUint, ref unmarshalRef, _ *core.GenContext) string {
	if v.Size > 64 {
		// limb-wise read into the array, unrolled with static byte ranges (the
		// slice machinery guarantees the source slice's exact width). An
		// overlay Cast cannot apply limb-wise, but overlay-of-wide-uint is
		// unreachable: go/types flattens a named wrapper's underlying type, so
		// a wrapper of uint256.Int never resolves to the wide uint
		// representation.
		lines := make([]string, int(v.Size)/64)
		for i := range lines {
			lines[i] = fmt.Sprintf("%s[%d] = binary.LittleEndian.Uint64(%s[%d:%d])", ref.FieldName, i, ref.SliceName, i*8, (i+1)*8)
		}
		return strings.Join(lines, "\n")
	}
	convert := fmt.Sprintf("binary.LittleEndian.Uint%d(%s)", v.Size, ref.SliceName)
	return fmt.Sprintf("%s = %s", ref.FieldName, ref.Cast(convert))
}

func (unmarshalOp) Bool(_ *gentypes.ValueBool, ref unmarshalRef, _ *core.GenContext) string {
	return execTmpl(unmarshalBoolTmpl, unmarshalCallElements{FieldName: ref.FieldName, SliceName: ref.SliceName})
}

func (unmarshalOp) Container(_ *gentypes.ValueContainer, ref unmarshalRef, _ *core.GenContext) string {
	return execTmpl(unmarshalCallTmpl, unmarshalCallElements{FieldName: ref.FieldName, SliceName: ref.SliceName, Field: ref.Field})
}

func (unmarshalOp) Pointer(v *gentypes.ValuePointer, ref unmarshalRef, ctx *core.GenContext) string {
	return core.Dispatch(unmarshalOp{}, v.Referent, ref, ctx)
}

func (unmarshalOp) Overlay(v *gentypes.ValueOverlay, ref unmarshalRef, ctx *core.GenContext) string {
	wrapper := ctx.QualifiedTypeName(v)
	child := ref
	child.Cast = func(s string) string { return fmt.Sprintf("%s(%s)", wrapper, s) }
	umv := core.Dispatch(unmarshalOp{}, v.Underlying, child, ctx)
	if v.IsBitfield() {
		if ul, ok := v.Underlying.(*gentypes.ValueList); ok {
			if ul.Progressive {
				return execTmpl(validateProgressiveBitlistTmpl, validateBitlistElements{SliceName: ref.SliceName, Unmarshal: umv, Field: ref.Field})
			}
			return execTmpl(validateBitlistTmpl, validateBitlistElements{SliceName: ref.SliceName, MaxSize: ul.MaxSize, Unmarshal: umv, Field: ref.Field})
		}
	}
	return umv
}

func (unmarshalOp) Vector(v *gentypes.ValueVector, ref unmarshalRef, ctx *core.GenContext) string {
	if _, isByte := v.ElementValue.(*gentypes.ValueByte); isByte {
		if v.IsArray {
			// array destination ([N]byte or a named array type): copy in
			// place, no allocation. The overlay Cast deliberately does not
			// apply — copy's source must be a slice, and a named array's
			// slice view needs no conversion.
			return fmt.Sprintf("copy(%s[:], %s)", ref.FieldName, ref.SliceName)
		}
		return execTmpl(unmarshalByteVectorTmpl, byteVectorCopyElements{
			FieldName: ref.FieldName,
			Size:      v.Size,
			Source:    ref.Cast(ref.SliceName),
		})
	}
	elem := v.ElementValue
	data := unmarshalLoopElements{
		FieldName:       ref.FieldName,
		TypeName:        ctx.QualifiedTypeName(elem),
		NumElements:     v.Size,
		LoopVar:         loopVarFor(ref.FieldName),
		Initializer:     elemInitializer(elem, ctx, tmpName(ref.Depth+1)),
		SliceName:       ref.SliceName,
		NestedFixedSize: elem.FixedSize(),
		Tmp:             tmpName(ref.Depth + 1),
		TmpSlice:        tmpSliceName(ref.Depth + 1),
		NestedUnmarshal: core.Dispatch(unmarshalOp{}, elem, unmarshalRef{FieldName: tmpName(ref.Depth + 1), SliceName: tmpSliceName(ref.Depth + 1), Cast: identity, Depth: ref.Depth + 1, Field: ref.Field}, ctx),
	}
	if elem.IsVariableSized() {
		// Offsets-then-payloads, like a variable list, but the element count is
		// part of the type: the first offset must be exactly 4*Size.
		return execTmpl(unmarshalVectorVariableTmpl, data)
	}
	return execTmpl(unmarshalVectorTmpl, data)
}

func (unmarshalOp) List(v *gentypes.ValueList, ref unmarshalRef, ctx *core.GenContext) string {
	elem := v.ElementValue
	if elem.IsVariableSized() {
		if v.Progressive {
			// progressive lists have no limit, hence no ssz-max check
			return execTmpl(unmarshalProgressiveListVariableTmpl, unmarshalListData(v, ref, ctx))
		}
		return execTmpl(unmarshalListVariableTmpl, unmarshalListData(v, ref, ctx))
	}
	if _, isByte := elem.(*gentypes.ValueByte); isByte {
		return fmt.Sprintf("%s = append([]byte{}, %s...)", ref.FieldName, ref.Cast(ref.SliceName))
	}
	if v.Progressive {
		return execTmpl(unmarshalProgressiveListFixedTmpl, unmarshalListData(v, ref, ctx))
	}
	return execTmpl(unmarshalListFixedTmpl, unmarshalListData(v, ref, ctx))
}

// unmarshalLoopElements parameterizes the element-decoding loops shared by the
// vector and both list templates. NumElements is vector-only (fixed element
// count); ElementSize and MaxSize are list-only. Tmp/TmpSlice are the
// depth-unique element temp names.
type unmarshalLoopElements struct {
	FieldName       string
	TypeName        string
	SliceName       string
	LoopVar         string
	Initializer     string
	NestedFixedSize int
	NestedUnmarshal string
	Tmp             string
	TmpSlice        string
	NumElements     int
	ElementSize     int
	MaxSize         int
}

func unmarshalListData(v *gentypes.ValueList, ref unmarshalRef, ctx *core.GenContext) unmarshalLoopElements {
	elem := v.ElementValue
	return unmarshalLoopElements{
		LoopVar:         loopVarFor(ref.FieldName),
		SliceName:       ref.SliceName,
		ElementSize:     elem.FixedSize(),
		TypeName:        ctx.QualifiedTypeName(elem),
		FieldName:       ref.FieldName,
		MaxSize:         v.MaxSize,
		Initializer:     elemInitializer(elem, ctx, tmpName(ref.Depth+1)),
		NestedFixedSize: elem.FixedSize(),
		Tmp:             tmpName(ref.Depth + 1),
		TmpSlice:        tmpSliceName(ref.Depth + 1),
		NestedUnmarshal: core.Dispatch(unmarshalOp{}, elem, unmarshalRef{FieldName: tmpName(ref.Depth + 1), SliceName: tmpSliceName(ref.Depth + 1), Cast: identity, Depth: ref.Depth + 1, Field: ref.Field}, ctx),
	}
}

func elemInitializer(elem gentypes.ValRep, ctx *core.GenContext, tmp string) string {
	ini := initValue(elem, ctx, ctx.Ifaces().Unmarshaler)
	if ini == "" {
		return ""
	}
	return tmp + " = " + ini
}

func loopVarFor(fieldName string) string {
	if fieldName[0:1] == "i" && monoCharacter(fieldName) {
		return fieldName + "i"
	}
	return "i"
}

// ---- fixed/variable slice machinery ----

func unmarshalFixedOffset(vc *gentypes.ValueContainer) int {
	offset := 0
	for _, c := range vc.Contents {
		offset += c.Value.FixedSize()
	}
	return offset
}

func sliceName(fieldNumber int) string {
	return fmt.Sprintf("sszSlice%d", fieldNumber)
}

type unmarshalStep struct {
	valRep           gentypes.ValRep
	fieldNumber      int
	fieldName        string
	beginByte        int
	endByte          int
	previousVariable *unmarshalStep
	nextVariable     *unmarshalStep
}

type unmarshalStepSlice []*unmarshalStep

func (us *unmarshalStep) varOffsetName() string { return fmt.Sprintf("sszVarOffset%d", us.fieldNumber) }
func (us *unmarshalStep) sliceName() string     { return sliceName(us.fieldNumber) }

func (us *unmarshalStep) variableOffset(outerFixedSize int) string {
	data := varOffsetElements{
		VarOffsetName: us.varOffsetName(),
		FieldName:     us.fieldName,
		Begin:         us.beginByte,
		End:           us.endByte,
		OuterSize:     outerFixedSize,
	}
	if us.previousVariable == nil {
		return execTmpl(unmarshalFirstOffsetTmpl, data)
	}
	data.PrevOffsetName = us.previousVariable.varOffsetName()
	return execTmpl(unmarshalNextOffsetTmpl, data)
}

// varOffsetElements parameterizes the offset validations: the offset variable
// being read, the byte range it is read from, and — for the first variable
// field — the container's fixed size, or — for subsequent fields — the previous
// field's offset variable.
type varOffsetElements struct {
	VarOffsetName  string
	FieldName      string
	Begin          int
	End            int
	OuterSize      int
	PrevOffsetName string
}

func (us *unmarshalStep) slice() string {
	if us.valRep.IsVariableSized() {
		if us.nextVariable == nil {
			return fmt.Sprintf("%s := buf[%s:]\t\t// %s", us.sliceName(), us.varOffsetName(), us.fieldName)
		}
		return fmt.Sprintf("%s := buf[%s:%s]\t\t// %s", us.sliceName(), us.varOffsetName(), us.nextVariable.varOffsetName(), us.fieldName)
	}
	return fmt.Sprintf("%s := buf[%d:%d]\t\t// %s", us.sliceName(), us.beginByte, us.endByte, us.fieldName)
}

func (steps unmarshalStepSlice) fixedSlices() string {
	slices := make([]string, 0)
	for _, s := range steps {
		if s.valRep.IsVariableSized() {
			continue
		}
		slices = append(slices, s.slice())
	}
	return strings.Join(slices, "\n")
}

func (steps unmarshalStepSlice) variableSlices(outerSize int) string {
	validate := make([]string, 0)
	assign := make([]string, 0)
	for _, s := range steps {
		if !s.valRep.IsVariableSized() {
			continue
		}
		validate = append(validate, s.variableOffset(outerSize))
		assign = append(assign, s.slice())
	}
	return strings.Join(append(validate, assign...), "\n")
}

func buildUnmarshalSteps(vc *gentypes.ValueContainer) unmarshalStepSlice {
	ums := make([]*unmarshalStep, 0)
	var begin, end int
	var prevVariable *unmarshalStep
	for i, c := range vc.Contents {
		begin = end
		end += c.Value.FixedSize()
		um := &unmarshalStep{
			valRep:      c.Value,
			fieldNumber: i,
			fieldName:   fmt.Sprintf("%s.%s", unmarshalReceiver, c.Key),
			beginByte:   begin,
			endByte:     end,
		}
		if c.Value.IsVariableSized() {
			if prevVariable != nil {
				um.previousVariable = prevVariable
				prevVariable.nextVariable = um
			}
			prevVariable = um
		}
		ums = append(ums, um)
	}
	return ums
}

// unmarshalCallElements parameterizes the fragments reading a field's value out
// of its source slice.
type unmarshalCallElements struct {
	FieldName string
	SliceName string
	Field     string
}

// validateBitlistElements parameterizes the bitlist guard wrapped around an
// unmarshal fragment.
type validateBitlistElements struct {
	SliceName string
	MaxSize   int
	Unmarshal string
	Field     string
}

// byteVectorCopyElements parameterizes the byte-vector copy: Source is the
// source slice expression, already cast by any enclosing overlay.
type byteVectorCopyElements struct {
	FieldName string
	Size      int
	Source    string
}

var (
	unmarshalCallTmpl = template.Must(template.New("unmarshalCall").Parse(
		`if err = {{.FieldName}}.UnmarshalSSZ({{.SliceName}}); err != nil {
	return fmt.Errorf("{{.Field}}: %w", err)
}`))

	unmarshalBoolTmpl = template.Must(template.New("unmarshalBool").Parse(
		`if {{.SliceName}}[0] > 1 {
	return ssz.ErrInvalidSerialization
}
if {{.SliceName}}[0] == 1 {
	{{.FieldName}} = true
} else {
	{{.FieldName}} = false
}`))

	validateBitlistTmpl = template.Must(template.New("validateBitlist").Parse(
		`if err = ssz.ValidateBitlist({{.SliceName}}, {{.MaxSize}}); err != nil {
	return fmt.Errorf("{{.Field}}: %w", err)
}
{{.Unmarshal}}`))

	unmarshalByteVectorTmpl = template.Must(template.New("unmarshalByteVector").Parse(
		`{{.FieldName}} = make([]byte, 0, {{.Size}})
{{.FieldName}} = append({{.FieldName}}, {{.Source}}...)`))

	unmarshalFirstOffsetTmpl = template.Must(template.New("unmarshalFirstOffset").Parse(
		`{{.VarOffsetName}} := ssz.ReadOffset(buf[{{.Begin}}:{{.End}}]) // {{.FieldName}}
if {{.VarOffsetName}} != {{.OuterSize}} {
	return ssz.ErrInvalidVariableOffset
}
if {{.VarOffsetName}} > size {
	return ssz.ErrOffset
}`))

	unmarshalNextOffsetTmpl = template.Must(template.New("unmarshalNextOffset").Parse(
		`{{.VarOffsetName}} := ssz.ReadOffset(buf[{{.Begin}}:{{.End}}]) // {{.FieldName}}
if {{.VarOffsetName}} > size || {{.VarOffsetName}} < {{.PrevOffsetName}} {
	return ssz.ErrOffset
}`))

	unmarshalVectorTmpl = template.Must(template.New("unmarshalVector").Parse(`{
	{{.FieldName}} = make([]{{.TypeName}}, {{.NumElements}})
	for {{.LoopVar}} := 0; {{.LoopVar}} < {{.NumElements}}; {{.LoopVar}} ++ {
		var {{.Tmp}} {{.TypeName}}
		{{.Initializer}}
		{{.TmpSlice}} := {{.SliceName}}[{{.LoopVar}}*{{.NestedFixedSize}}:(1+{{.LoopVar}})*{{.NestedFixedSize}}]
{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = {{.Tmp}}
	}
}`))

	// A vector of variable-sized elements decodes like a variable list, but the
	// element count is part of the type: there are exactly NumElements offsets,
	// so the first offset must equal 4*NumElements.
	unmarshalVectorVariableTmpl = template.Must(template.New("unmarshalVectorVariable").Parse(`{
	if len({{.SliceName}}) < 4*{{.NumElements}} {
		return fmt.Errorf("vector bytes too short to contain offsets when decoding {{.FieldName}}: %w", ssz.ErrSize)
	}
	startOffset := ssz.ReadOffset({{.SliceName}}[0:4])
	if startOffset != 4*{{.NumElements}} {
		return fmt.Errorf("invalid initial offset %d when decoding {{.FieldName}}, expected %d: %w", startOffset, 4*{{.NumElements}}, ssz.ErrInvalidVariableOffset)
	}
	totalVarBytes := uint64(len({{.SliceName}}))
	{{.FieldName}} = make([]{{.TypeName}}, {{.NumElements}})
	var {{.TmpSlice}} []byte
	for {{.LoopVar}} := uint64(0); {{.LoopVar}} < {{.NumElements}}; {{.LoopVar}}++ {
		var {{.Tmp}} {{.TypeName}}
		{{.Initializer}}
		endOffset := totalVarBytes
		if {{.LoopVar}}+1 != {{.NumElements}} {
			endOffset = ssz.ReadOffset({{.SliceName}}[({{.LoopVar}}+1)*4:({{.LoopVar}}+2)*4])
			if totalVarBytes < endOffset {
				return fmt.Errorf("offset %d points past the end of buffer when decoding {{.FieldName}}", endOffset)
			}
		}
		if endOffset < startOffset {
			return fmt.Errorf("offset %d is not greater than start offset %d when decoding {{.FieldName}}", endOffset, startOffset)
		}
		{{.TmpSlice}} = {{.SliceName}}[startOffset:endOffset]
		{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = {{.Tmp}}
		startOffset = endOffset
	}
}`))

	unmarshalListFixedTmpl = template.Must(template.New("unmarshalListFixed").Parse(`{
	if len({{.SliceName}}) % {{.ElementSize}} != 0 {
		return fmt.Errorf("misaligned bytes: {{.FieldName}} length is %d, which is not a multiple of {{.ElementSize}}: %w", len({{.SliceName}}), ssz.ErrIncorrectListSize)
	}
	numElem := len({{.SliceName}}) / {{.ElementSize}}
	if numElem > {{.MaxSize}} {
		return fmt.Errorf("ssz-max exceeded: {{.FieldName}} has %d elements, ssz-max is {{.MaxSize}}: %w", numElem, ssz.ErrListTooBig)
	}
	{{.FieldName}} = make([]{{.TypeName}}, numElem)
	for {{.LoopVar}} := 0; {{.LoopVar}} < numElem; {{.LoopVar}}++ {
		var {{.Tmp}} {{.TypeName}}
		{{.Initializer}}
		{{.TmpSlice}} := {{.SliceName}}[{{.LoopVar}}*{{.NestedFixedSize}}:(1+{{.LoopVar}})*{{.NestedFixedSize}}]
	{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = {{.Tmp}}
	}
}`))

	unmarshalListVariableTmpl = template.Must(template.New("unmarshalListVariable").Parse(`{
// empty lists are zero length, so make sure there is room for an offset
// before attempting to unmarshal it
if len({{.SliceName}}) > 3 {
	startOffset := ssz.ReadOffset({{.SliceName}}[0:4])
	if startOffset == 0 {
		return fmt.Errorf("encountered invalid offset of 0 when decoding {{.FieldName}}")
	}
	if startOffset % 4 != 0 {
		return fmt.Errorf("misaligned list bytes: when decoding {{.FieldName}}, end-of-list offset is %d, which is not a multiple of 4 (offset size)", startOffset)
	}
	listLen := startOffset / 4
	if listLen > {{.MaxSize}} {
			return fmt.Errorf("ssz-max exceeded: {{.FieldName}} has %d elements, ssz-max is {{.MaxSize}}: %w", listLen, ssz.ErrListTooBig)
	}
	totalVarBytes := uint64(len({{.SliceName}}))
	if totalVarBytes < startOffset {
		return fmt.Errorf("list bytes too short to contain an offset when decoding {{.FieldName}}")
	}
	{{.FieldName}} = make([]{{.TypeName}}, listLen)
	var {{.TmpSlice}} []byte
	for {{.LoopVar}} := uint64(0); {{.LoopVar}} < listLen; {{.LoopVar}}++ {
		var {{.Tmp}} {{.TypeName}}
		{{.Initializer}}
		endOffset := totalVarBytes
		if {{.LoopVar}}+1 != listLen {
			endOffset = ssz.ReadOffset({{.SliceName}}[({{.LoopVar}}+1)*4:({{.LoopVar}}+2)*4])
			if totalVarBytes < endOffset {
				return fmt.Errorf("offset %d points past the end of buffer when decoding {{.FieldName}}", endOffset)
			}
		}
		if endOffset < startOffset {
			return fmt.Errorf("offset %d is not greater than start offset %d when decoding {{.FieldName}}", endOffset, startOffset)
		}
		{{.TmpSlice}} = {{.SliceName}}[startOffset:endOffset]
		{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = {{.Tmp}}
		startOffset = endOffset
	}
} else {
	if len({{.SliceName}}) > 0 {
		return fmt.Errorf("list bytes too short to contain an offset when decoding {{.FieldName}}")
	}
	{{.FieldName}} = make([]{{.TypeName}}, 0)
}
}`))
)

// Progressive list variants: same decoding loops, but no ssz-max check
// (progressive lists are unlimited).
var (
	validateProgressiveBitlistTmpl = template.Must(template.New("validateProgressiveBitlist").Parse(
		`if err = ssz.ValidateProgressiveBitlist({{.SliceName}}); err != nil {
	return fmt.Errorf("{{.Field}}: %w", err)
}
{{.Unmarshal}}`))

	unmarshalProgressiveListFixedTmpl = template.Must(template.New("unmarshalProgressiveListFixed").Parse(`{
	if len({{.SliceName}}) % {{.ElementSize}} != 0 {
		return fmt.Errorf("misaligned bytes: {{.FieldName}} length is %d, which is not a multiple of {{.ElementSize}}: %w", len({{.SliceName}}), ssz.ErrIncorrectListSize)
	}
	numElem := len({{.SliceName}}) / {{.ElementSize}}
	{{.FieldName}} = make([]{{.TypeName}}, numElem)
	for {{.LoopVar}} := 0; {{.LoopVar}} < numElem; {{.LoopVar}}++ {
		var {{.Tmp}} {{.TypeName}}
		{{.Initializer}}
		{{.TmpSlice}} := {{.SliceName}}[{{.LoopVar}}*{{.NestedFixedSize}}:(1+{{.LoopVar}})*{{.NestedFixedSize}}]
	{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = {{.Tmp}}
	}
}`))

	unmarshalProgressiveListVariableTmpl = template.Must(template.New("unmarshalProgressiveListVariable").Parse(`{
// empty lists are zero length, so make sure there is room for an offset
// before attempting to unmarshal it
if len({{.SliceName}}) > 3 {
	startOffset := ssz.ReadOffset({{.SliceName}}[0:4])
	if startOffset == 0 {
		return fmt.Errorf("encountered invalid offset of 0 when decoding {{.FieldName}}")
	}
	if startOffset % 4 != 0 {
		return fmt.Errorf("misaligned list bytes: when decoding {{.FieldName}}, end-of-list offset is %d, which is not a multiple of 4 (offset size)", startOffset)
	}
	listLen := startOffset / 4
	totalVarBytes := uint64(len({{.SliceName}}))
	if totalVarBytes < startOffset {
		return fmt.Errorf("list bytes too short to contain an offset when decoding {{.FieldName}}")
	}
	{{.FieldName}} = make([]{{.TypeName}}, listLen)
	var {{.TmpSlice}} []byte
	for {{.LoopVar}} := uint64(0); {{.LoopVar}} < listLen; {{.LoopVar}}++ {
		var {{.Tmp}} {{.TypeName}}
		{{.Initializer}}
		endOffset := totalVarBytes
		if {{.LoopVar}}+1 != listLen {
			endOffset = ssz.ReadOffset({{.SliceName}}[({{.LoopVar}}+1)*4:({{.LoopVar}}+2)*4])
			if totalVarBytes < endOffset {
				return fmt.Errorf("offset %d points past the end of buffer when decoding {{.FieldName}}", endOffset)
			}
		}
		if endOffset < startOffset {
			return fmt.Errorf("offset %d is not greater than start offset %d when decoding {{.FieldName}}", endOffset, startOffset)
		}
		{{.TmpSlice}} = {{.SliceName}}[startOffset:endOffset]
		{{.NestedUnmarshal}}
		{{.FieldName}}[{{.LoopVar}}] = {{.Tmp}}
		startOffset = endOffset
	}
} else {
	if len({{.SliceName}}) > 0 {
		return fmt.Errorf("list bytes too short to contain an offset when decoding {{.FieldName}}")
	}
	{{.FieldName}} = make([]{{.TypeName}}, 0)
}
}`))
)
