package sszgen

import (
	"reflect"
	"testing"

	_ "github.com/OffchainLabs/methodical-ssz/sszgen/testdata"
	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

var (
	packageName = "github.com/OffchainLabs/methodical-ssz/sszgen/testdata"
	noImports   = "NoImports"
)

func TestGetSimpleRepresentation(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatal(err)
	}
	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatal(err)
	}
	for _, td := range defs {
		_, err := ParseTypeDef(td)
		if err != nil {
			t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
		}
	}
}

// TestSimpleStructRepresentation ensures that a type declaration like:
// type AliasedPrimitive uint64
// will be represented like ValueOverlay{Name: "AliasedPrimitive", Underlying: ValueUint{Name: "uint64"}}
func TestPrimitiveAliasRepresentation(t *testing.T) {
	typeName := "AliasedPrimitive"
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}
	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	for _, td := range defs {
		val, err := ParseTypeDef(td)
		if err != nil {
			t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
		}
		if val.TypeName() != typeName {
			t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
		}
		overlay, ok := val.(*types.ValueOverlay)
		if !ok {
			t.Fatal("type declaration over primitive type should result in a ValueOverlay")
		}
		underlyingTypeName := overlay.Underlying.TypeName()
		if underlyingTypeName != "uint64" {
			t.Fatalf("expected underlying type name uint64, got %s", underlyingTypeName)
		}
	}
}

func TestSimpleStructRepresentation(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	// test simple "overlay" values
	overlayValRep, err := container.GetField("MuhPrim")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	overlay, ok := overlayValRep.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected the result to be a ValueOverlay type, got %v", typename(overlayValRep))
	}
	if overlay.TypeName() != "AliasedPrimitive" {
		t.Fatalf("expected overlay type name AliasedPrimitive, got %s", overlay.TypeName())
	}
	if overlay.Underlying.TypeName() != "uint64" {
		t.Fatalf("expected underlying type name uint64, got %s", overlay.Underlying.TypeName())
	}

	uintValRep, err := container.GetField("GenesisTime")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if !ok {
		t.Fatal("Expected \"GenesisTime\" to be in container")
	}
	if uintValRep.TypeName() != "uint64" {
		t.Fatalf("expected type name uint64, got %s", uintValRep.TypeName())
	}
	uintType, ok := uintValRep.(*types.ValueUint)
	if !ok {
		t.Fatalf("Expected \"GenesisTime\" to be a ValueUint, got %v", typename(uintValRep))
	}
	if uintType.Size != types.UintSize(64) {
		t.Fatalf("expected size 64, got %d", uintType.Size)
	}
}

// Tests that 1 and 2 dimensional vectors are represented as expected
func TestStructVectors(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	vectorValRep, err := container.GetField("GenesisValidatorsRoot")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if !ok {
		t.Fatal("Expected \"GenesisValidatorsRoot\" to be in container")
	}
	vector, ok := vectorValRep.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueVector type, got %v", typename(vectorValRep))
	}
	if vector.TypeName() != "[]byte" {
		t.Fatalf("expected type name []byte, got %s", vector.TypeName())
	}
	byteVal, ok := vector.ElementValue.(*types.ValueByte)
	if !ok {
		t.Fatalf("Expected the ElementValue a ValueByte type, got %v", typename(vector))
	}
	if byteVal.TypeName() != "byte" {
		t.Fatalf("expected type name byte, got %s", byteVal.TypeName())
	}
	if vector.Size != 32 {
		t.Fatalf("expected size 32, got %d", vector.Size)
	}

	vectorValRep2d, err := container.GetField("BlockRoots")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	vector2d, ok := vectorValRep2d.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueVector type, got %v", typename(vectorValRep2d))
	}
	if vector2d.Size != 8192 {
		t.Fatalf("expected size 8192, got %d", vector2d.Size)
	}
	vector1d, ok := vector2d.ElementValue.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the element type of \"BlockRoots\" to be type ValueVector, got %v", typename(vector2d.ElementValue))
	}
	if vector1d.Size != 32 {
		t.Fatalf("expected size 32, got %d", vector1d.Size)
	}
	vector1dElement, ok := vector1d.ElementValue.(*types.ValueByte)
	if !ok {
		t.Fatalf("Expected the element type of \"BlockRoots\" to be type ValueVector, got %v", typename(vector2d.ElementValue))
	}
	if vector1dElement.TypeName() != "byte" {
		t.Fatalf("expected type name byte, got %s", vector1dElement.TypeName())
	}
}

// tests that ssz dimensions are assigned correctly with a vector nested in a list
func TestVectorInListInStruct(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	listValRep, err := container.GetField("HistoricalRoots")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if listValRep.TypeName() != "[][]byte" {
		t.Fatalf("expected type name [][]byte, got %s", listValRep.TypeName())
	}
	list, ok := listValRep.(*types.ValueList)
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(listValRep))
	}
	if list.MaxSize != 16777216 {
		t.Fatalf("Unexpected value for list max size based on parsed ssz tags, want 16777216, got %d", list.MaxSize)
	}

	if list.ElementValue.TypeName() != "[]byte" {
		t.Fatalf("expected type name []byte, got %s", list.ElementValue.TypeName())
	}
	vector, ok := list.ElementValue.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueVector type, got %v", typename(list.ElementValue))
	}
	if vector.Size != 32 {
		t.Fatalf("expected size 32, got %d", vector.Size)
	}

	if vector.ElementValue.TypeName() != "byte" {
		t.Fatalf("expected type name byte, got %s", vector.ElementValue.TypeName())
	}
	_, ok = vector.ElementValue.(*types.ValueByte)
	if !ok {
		t.Fatalf("Expected the ElementValue a ValueByte type, got %v", typename(vector.ElementValue))

	}
}

func TestContainerField(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	fieldValRep, err := container.GetField("ContainerField")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if fieldValRep.TypeName() != "ContainerType" {
		t.Fatalf("expected type name ContainerType, got %s", fieldValRep.TypeName())
	}
	field, ok := fieldValRep.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(fieldValRep))
	}
	if len(field.Fields()) != 1 {
		t.Fatalf("expected 1 field in ContainerType, got %d", len(field.Fields()))
	}

	refFieldValRep, err := container.GetField("ContainerRefField")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if refFieldValRep.TypeName() != "*AnotherContainerType" {
		t.Fatalf("expected type name *AnotherContainerType, got %s", refFieldValRep.TypeName())
	}
	refField, ok := refFieldValRep.(*types.ValuePointer)
	if !ok {
		t.Fatalf("Expected the result to be a ValuePointer type, got %v", typename(refFieldValRep))
	}
	cont, isCont := refField.Referent.(*types.ValueContainer)
	if !isCont {
		t.Fatalf("Expected the referent of ContainerRefField to be a ValueContainer, got %v", typename(refField.Referent))
	}
	if len(cont.Fields()) != 1 {
		t.Fatalf("expected 1 field in AnotherContainerType, got %d", len(cont.Fields()))
	}
}

func TestListContainers(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	conlistValRep, err := container.GetField("ContainerList")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if conlistValRep.TypeName() != "[]ContainerType" {
		t.Fatalf("expected type name []ContainerType, got %s", conlistValRep.TypeName())
	}
	conlist, ok := conlistValRep.(*types.ValueList)
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(conlistValRep))

	}
	if conlist.MaxSize != 23 {
		t.Fatalf("expected max size 23, got %d", conlist.MaxSize)
	}
	if conlist.ElementValue.TypeName() != "ContainerType" {
		t.Fatalf("expected type name ContainerType, got %s", conlist.ElementValue.TypeName())
	}

	conVecValRep, err := container.GetField("ContainerVector")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if conVecValRep.TypeName() != "[]ContainerType" {
		t.Fatalf("expected type name []ContainerType, got %s", conVecValRep.TypeName())
	}
	conVec, ok := conVecValRep.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueVector type, got %v", typename(conVecValRep))
	}
	if conVec.Size != 42 {
		t.Fatalf("expected size 42, got %d", conVec.Size)
	}
	if conVec.ElementValue.TypeName() != "ContainerType" {
		t.Fatalf("expected type name ContainerType, got %s", conVec.ElementValue.TypeName())
	}

	conVecValRefRep, err := container.GetField("ContainerVectorRef")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if conVecValRefRep.TypeName() != "[]*ContainerType" {
		t.Fatalf("expected type name []*ContainerType, got %s", conVecValRefRep.TypeName())
	}
	conVecRef, ok := conVecValRefRep.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueVector type, got %v", typename(conVecValRefRep))
	}
	conVecRefPointer, ok := conVecRef.ElementValue.(*types.ValuePointer)
	if !ok {
		t.Fatalf("Expected the result to be a ValuePointer type, got %v", typename(conVecRef.ElementValue))
	}
	conVecReferent, ok := conVecRefPointer.Referent.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(conVecRefPointer.Referent))
	}
	if conVecReferent.TypeName() != "ContainerType" {
		t.Fatalf("expected type name ContainerType, got %s", conVecReferent.TypeName())
	}
	if conVecRef.Size != 17 {
		t.Fatalf("expected size 17, got %d", conVecRef.Size)
	}

	conListValRefRep, err := container.GetField("ContainerListRef")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if conListValRefRep.TypeName() != "[]*ContainerType" {
		t.Fatalf("expected type name []*ContainerType, got %s", conListValRefRep.TypeName())
	}
	conListRef, ok := conListValRefRep.(*types.ValueList)
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(conListValRefRep))
	}
	conListRefPointer, ok := conListRef.ElementValue.(*types.ValuePointer)
	if !ok {
		t.Fatalf("Expected the result to be a ValuePointer type, got %v", typename(conListRef.ElementValue))
	}
	conListReferent, ok := conListRefPointer.Referent.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(conListRefPointer.Referent))
	}
	if conListReferent.TypeName() != "ContainerType" {
		t.Fatalf("expected type name ContainerType, got %s", conListReferent.TypeName())
	}
	if conListRef.MaxSize != 9000 {
		t.Fatalf("expected max size 9000, got %d", conListRef.MaxSize)
	}
}

func TestListOfOverlays(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}

	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	overlayListRep, err := container.GetField("OverlayList")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if overlayListRep.TypeName() != "[]AliasedPrimitive" {
		t.Fatalf("expected type name []AliasedPrimitive, got %s", overlayListRep.TypeName())
	}
	overlayList, ok := overlayListRep.(*types.ValueList)
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(overlayListRep))
	}
	if overlayList.MaxSize != 11 {
		t.Fatalf("expected max size 11, got %d", overlayList.MaxSize)
	}
	if overlayList.ElementValue.TypeName() != "AliasedPrimitive" {
		t.Fatalf("expected type name AliasedPrimitive, got %s", overlayList.ElementValue.TypeName())
	}
	overlay, ok := overlayList.ElementValue.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected a ValueOverly, got %v", typename(overlayList.ElementValue))
	}
	if overlay.Underlying.TypeName() != "uint64" {
		t.Fatalf("expected underlying type name uint64, got %s", overlay.Underlying.TypeName())
	}

	underlying, ok := overlay.Underlying.(*types.ValueUint)
	if !ok {
		t.Fatalf("Expected a ValueUint, got %v", typename(overlay.Underlying))
	}
	if underlying.Size != types.UintSize(64) {
		t.Fatalf("expected size 64, got %d", underlying.Size)
	}

	overlayListRefRep, err := container.GetField("OverlayListRef")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if overlayListRefRep.TypeName() != "[]*AliasedPrimitive" {
		t.Fatalf("expected type name []*AliasedPrimitive, got %s", overlayListRefRep.TypeName())
	}
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(overlayListRefRep))
	}
	overlayRefList, ok := overlayListRefRep.(*types.ValueList)
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(overlayRefList))
	}
	if overlayRefList.MaxSize != 58 {
		t.Fatalf("expected max size 58, got %d", overlayRefList.MaxSize)
	}
	if overlayRefList.ElementValue.TypeName() != "*AliasedPrimitive" {
		t.Fatalf("expected type name *AliasedPrimitive, got %s", overlayRefList.ElementValue.TypeName())
	}
	overlayPointer, ok := overlayRefList.ElementValue.(*types.ValuePointer)
	if !ok {
		t.Fatalf("Expected a ValuePointer, got %v", typename(overlayRefList.ElementValue))
	}
	if overlayPointer.Referent.TypeName() != "AliasedPrimitive" {
		t.Fatalf("expected referent type name AliasedPrimitive, got %s", overlayPointer.Referent.TypeName())
	}
	overlayRef, ok := overlayPointer.Referent.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected a ValueOverlay, got %v", typename(overlayPointer.Referent))
	}
	if overlayRef.Underlying.TypeName() != "uint64" {
		t.Fatalf("expected underlying type name uint64, got %s", overlayRef.Underlying.TypeName())
	}
	underlyingRef, ok := overlay.Underlying.(*types.ValueUint)
	if !ok {
		t.Fatalf("Expected a ValueUint, got %v", typename(overlayRef.Underlying))
	}
	if underlyingRef.Size != types.UintSize(64) {
		t.Fatalf("expected size 64, got %d", underlyingRef.Size)
	}
}

func TestVectorOfOverlays(t *testing.T) {
	typeName := noImports
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}

	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	overlayVectorRep, err := container.GetField("OverlayVector")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if overlayVectorRep.TypeName() != "[]AliasedPrimitive" {
		t.Fatalf("expected type name []AliasedPrimitive, got %s", overlayVectorRep.TypeName())
	}
	overlayVector, ok := overlayVectorRep.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected a ValueVector, got %v", typename(overlayVectorRep))
	}
	if overlayVector.Size != 23 {
		t.Fatalf("expected size 23, got %d", overlayVector.Size)
	}
	if overlayVector.ElementValue.TypeName() != "AliasedPrimitive" {
		t.Fatalf("expected type name AliasedPrimitive, got %s", overlayVector.ElementValue.TypeName())
	}
	overlay, ok := overlayVector.ElementValue.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected a ValueOverly, got %v", typename(overlayVector.ElementValue))
	}
	if overlay.Underlying.TypeName() != "uint64" {
		t.Fatalf("expected underlying type name uint64, got %s", overlay.Underlying.TypeName())
	}
	underlying, ok := overlay.Underlying.(*types.ValueUint)
	if !ok {
		t.Fatalf("Expected a ValueUint, got %v", typename(overlay.Underlying))

	}
	if underlying.Size != types.UintSize(64) {
		t.Fatalf("expected size 64, got %d", underlying.Size)
	}

	overlayVectorRefRep, err := container.GetField("OverlayVectorRef")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	if overlayVectorRefRep.TypeName() != "[]*AliasedPrimitive" {
		t.Fatalf("expected type name []*AliasedPrimitive, got %s", overlayVectorRefRep.TypeName())
	}
	overlayRefVector, ok := overlayVectorRefRep.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected a ValueVector, got %v", typename(overlayRefVector))
	}
	if overlayRefVector.Size != 13 {
		t.Fatalf("expected size 13, got %d", overlayRefVector.Size)
	}
	if overlayRefVector.ElementValue.TypeName() != "*AliasedPrimitive" {
		t.Fatalf("expected type name *AliasedPrimitive, got %s", overlayRefVector.ElementValue.TypeName())
	}
	overlayPointer, ok := overlayRefVector.ElementValue.(*types.ValuePointer)
	if !ok {
		t.Fatalf("Expected a ValuePointer, got %v", typename(overlayRefVector.ElementValue))

	}
	if overlayPointer.Referent.TypeName() != "AliasedPrimitive" {
		t.Fatalf("expected referent type name AliasedPrimitive, got %s", overlayPointer.Referent.TypeName())
	}
	overlayRef, ok := overlayPointer.Referent.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected a ValueOverlay, got %v", typename(overlayPointer.Referent))
	}
	if overlayRef.Underlying.TypeName() != "uint64" {
		t.Fatalf("expected underlying type name uint64, got %s", overlayRef.Underlying.TypeName())
	}
	underlyingRef, ok := overlay.Underlying.(*types.ValueUint)
	if !ok {
		t.Fatalf("Expected a ValueUint, got %v", typename(overlayRef.Underlying))
	}
	if underlyingRef.Size != types.UintSize(64) {
		t.Fatalf("expected size 64, got %d", underlyingRef.Size)
	}
}

func TestBitlist(t *testing.T) {
	typeName := "TestBitlist"
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewGoPathScoper=%s", err.Error())
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatalf("unexpected error from TypeDefs=%s", err.Error())
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}
	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatalf("unexpected error from ParseTypeDef=%s", err.Error())
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueContainer)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	overlayValRep, err := container.GetField("AggregationBits")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	overlay, ok := overlayValRep.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected the result to be a ValueOverlay type, got %v", typename(overlayValRep))
	}
	if overlay.TypeName() != "Bitlist" {
		t.Fatalf("expected type name Bitlist, got %s", overlay.TypeName())
	}
	if overlay.Underlying.TypeName() != "[]byte" {
		t.Fatalf("expected underlying type name []byte, got %s", overlay.Underlying.TypeName())
	}
	underlying, ok := overlay.Underlying.(*types.ValueList)
	if !ok {
		t.Fatalf("Expected the result to be a ValueList type, got %v", typename(overlayValRep))
	}
	if underlying.MaxSize != 2048 {
		t.Fatalf("expected max size 2048, got %d", underlying.MaxSize)
	}
	if underlying.ElementValue.TypeName() != "byte" {
		t.Fatalf("expected element type name byte, got %s", underlying.ElementValue.TypeName())
	}
	_, ok = underlying.ElementValue.(*types.ValueByte)
	if !ok {
		t.Fatalf("Expected the result to be a ValueByte type, got %v", typename(underlying.ElementValue))
	}

	overlayVecValRep, err := container.GetField("JustificationBits")
	if err != nil {
		t.Fatalf("unexpected error from GetField=%s", err.Error())
	}
	overlayVec, ok := overlayVecValRep.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected the result to be a ValueOverlay type, got %v", typename(overlayVecValRep))
	}
	if overlayVec.TypeName() != "Bitvector4" {
		t.Fatalf("expected type name Bitvector4, got %s", overlayVec.TypeName())
	}
	if overlayVec.Underlying.TypeName() != "[]byte" {
		t.Fatalf("expected underlying type name []byte, got %s", overlayVec.Underlying.TypeName())
	}
	underlyingVec, ok := overlayVec.Underlying.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueVector type, got %v", typename(overlayVec.Underlying))
	}
	if underlyingVec.Size != 1 {
		t.Fatalf("expected size 1, got %d", underlyingVec.Size)
	}

	if underlyingVec.ElementValue.TypeName() != "byte" {
		t.Fatalf("expected element type name byte, got %s", underlyingVec.ElementValue.TypeName())
	}
	_, ok = underlyingVec.ElementValue.(*types.ValueByte)
	if !ok {
		t.Fatalf("Expected the result to be a ValueByte type, got %v", typename(underlyingVec.ElementValue))
	}
}

func TestFixedSizeArray(t *testing.T) {
	typeName := "FixedSizeArray"
	ps, err := NewGoPathScoper(packageName, nil)
	if err != nil {
		t.Fatal(err)
	}

	defs, err := TypeDefs(ps, typeName)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 type definition, got %d", len(defs))
	}

	val, err := ParseTypeDef(defs[0])
	if err != nil {
		t.Fatal(err)
	}
	if val.TypeName() != typeName {
		t.Fatalf("expected type name %s, got %s", typeName, val.TypeName())
	}
	container, ok := val.(*types.ValueOverlay)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(container))
	}

	underlying := container.Underlying
	c2, ok := underlying.(*types.ValueVector)
	if !ok {
		t.Fatalf("Expected the result to be a ValueContainer type, got %v", typename(c2))
	}
	if !c2.IsArray {
		t.Fatal("Expected the result to be an array")
	}
}

func typename(v interface{}) string {
	ty := reflect.TypeOf(v)
	if ty.Kind() == reflect.Ptr {
		return "*" + ty.Elem().Name()
	} else {
		return ty.Name()
	}
}
