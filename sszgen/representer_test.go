package sszgen

import (
	"reflect"
	"testing"

	_ "github.com/OffchainLabs/methodical-ssz/sszgen/testdata"
	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

var (
	packageName = "github.com/OffchainLabs/methodical-ssz/sszgen/testdata"
	noImports   = "NoImports"
)

func TestGetSimpleRepresentation(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)
	for _, td := range pp.results {
		_, err := ParseTypeDef(td)
		require.NoError(t, err)
	}
}

// TestSimpleStructRepresentation ensures that a type declaration like:
// type AliasedPrimitive uint64
// will be represented like ValueOverlay{Name: "AliasedPrimitive", Underlying: ValueUint{Name: "uint64"}}
func TestPrimitiveAliasRepresentation(t *testing.T) {
	typeName := "AliasedPrimitive"
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)
	for _, td := range pp.results {
		val, err := ParseTypeDef(td)
		require.NoError(t, err)
		require.Equal(t, typeName, val.TypeName())
		overlay, ok := val.(*types.ValueOverlay)
		require.Equal(t, true, ok, "type declaration over primitive type should result in a ValueOverlay")
		require.Equal(t, "uint64", overlay.Underlying.TypeName())
	}
}

func TestSimpleStructRepresentation(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	// test simple "overlay" values
	overlayValRep, err := container.GetField("MuhPrim")
	require.NoError(t, err)
	overlay, ok := overlayValRep.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected the result to be a ValueOverlay type, got %v", typename(overlayValRep))
	require.Equal(t, "AliasedPrimitive", overlay.TypeName())
	require.Equal(t, "uint64", overlay.Underlying.TypeName())

	uintValRep, err := container.GetField("GenesisTime")
	require.NoError(t, err)
	require.Equal(t, true, ok, "Expected \"GenesisTime\" to be in container")
	require.Equal(t, "uint64", uintValRep.TypeName())
	uintType, ok := uintValRep.(*types.ValueUint)
	require.Equal(t, true, ok, "Expected \"GenesisTime\" to be a ValueUint, got %v", typename(uintValRep))
	require.Equal(t, types.UintSize(64), uintType.Size)
}

// Tests that 1 and 2 dimensional vectors are represented as expected
func TestStructVectors(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	vectorValRep, err := container.GetField("GenesisValidatorsRoot")
	require.NoError(t, err)
	require.Equal(t, true, ok, "Expected \"GenesisValidatorsRoot\" to be in container")
	vector, ok := vectorValRep.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the result to be a ValueVector type, got %v", typename(vectorValRep))
	require.Equal(t, "[]byte", vector.TypeName())
	byteVal, ok := vector.ElementValue.(*types.ValueByte)
	require.Equal(t, true, ok, "Expected the ElementValue a ValueByte type, got %v", typename(vector))
	require.Equal(t, byteVal.TypeName(), "byte")
	require.Equal(t, 32, vector.Size)

	vectorValRep2d, err := container.GetField("BlockRoots")
	require.NoError(t, err)
	vector2d, ok := vectorValRep2d.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected \"BlockRoots\" to be type ValueVector, got %v", typename(vector2d))
	require.Equal(t, 8192, vector2d.Size)
	vector1d, ok := vector2d.ElementValue.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the element type of \"BlockRoots\" to be type ValueVector, got %v", typename(vector1d))
	require.Equal(t, 32, vector1d.Size)
	vector1dElement, ok := vector1d.ElementValue.(*types.ValueByte)
	require.Equal(t, true, ok, "Expected the element type of \"BlockRoots\" to be type ValueVector, got %v", typename(vector2d.ElementValue))
	require.Equal(t, "byte", vector1dElement.TypeName())
}

// tests that ssz dimensions are assigned correctly with a vector nested in a list
func TestVectorInListInStruct(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	listValRep, err := container.GetField("HistoricalRoots")
	require.NoError(t, err)
	require.Equal(t, "[][]byte", listValRep.TypeName())
	list, ok := listValRep.(*types.ValueList)
	require.Equal(t, true, ok, "Expected the result to be a ValueOverlay type, got %v", typename(listValRep))
	require.Equal(t, 16777216, list.MaxSize, "Unexpected value for list max size based on parsed ssz tags")

	require.Equal(t, "[]byte", list.ElementValue.TypeName())
	vector, ok := list.ElementValue.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the result to be a ValueVector type, got %v", typename(list.ElementValue))
	require.Equal(t, 32, vector.Size)

	require.Equal(t, "byte", vector.ElementValue.TypeName())
	_, ok = vector.ElementValue.(*types.ValueByte)
	require.Equal(t, true, ok, "Expected the ElementValue a ValueByte type, got %v", typename(vector))
}

func TestContainerField(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	fieldValRep, err := container.GetField("ContainerField")
	require.NoError(t, err)
	require.Equal(t, "ContainerType", fieldValRep.TypeName())
	field, ok := fieldValRep.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(fieldValRep))
	require.Equal(t, 1, len(field.Fields()))

	refFieldValRep, err := container.GetField("ContainerRefField")
	require.NoError(t, err)
	require.Equal(t, "*AnotherContainerType", refFieldValRep.TypeName())
	refField, ok := refFieldValRep.(*types.ValuePointer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(refField))
	cont, isCont := refField.Referent.(*types.ValueContainer)
	require.Equal(t, true, isCont)
	require.Equal(t, 1, len(cont.Fields()))
}

func TestListContainers(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	conlistValRep, err := container.GetField("ContainerList")
	require.NoError(t, err)
	require.Equal(t, "[]ContainerType", conlistValRep.TypeName())
	conlist, ok := conlistValRep.(*types.ValueList)
	require.Equal(t, true, ok, "Expected the result to be a ValueListtype, got %v", typename(conlistValRep))
	require.Equal(t, 23, conlist.MaxSize)
	require.Equal(t, "ContainerType", conlist.ElementValue.TypeName())

	conVecValRep, err := container.GetField("ContainerVector")
	require.NoError(t, err)
	require.Equal(t, "[]ContainerType", conVecValRep.TypeName())
	conVec, ok := conVecValRep.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the result to be a ValueVec, got %v", typename(conVecValRep))
	require.Equal(t, 42, conVec.Size)
	require.Equal(t, "ContainerType", conVec.ElementValue.TypeName())

	conVecValRefRep, err := container.GetField("ContainerVectorRef")
	require.NoError(t, err)
	require.Equal(t, "[]*ContainerType", conVecValRefRep.TypeName())
	conVecRef, ok := conVecValRefRep.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the result to be a ValueVector, got %v", typename(conVecValRefRep))
	conVecRefPointer, ok := conVecRef.ElementValue.(*types.ValuePointer) //
	require.Equal(t, true, ok, "Expected the result to be a ValuePointer, got %v", typename(conVecRef.ElementValue))
	conVecReferent, ok := conVecRefPointer.Referent.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer, got %v", typename(conVecRefPointer.Referent))
	require.Equal(t, "ContainerType", conVecReferent.TypeName())
	require.Equal(t, 17, conVecRef.Size)

	conListValRefRep, err := container.GetField("ContainerListRef")
	require.NoError(t, err)
	require.Equal(t, "[]*ContainerType", conListValRefRep.TypeName())
	conListRef, ok := conListValRefRep.(*types.ValueList)
	require.Equal(t, true, ok, "Expected the result to be a ValueList, got %v", typename(conListValRefRep))
	conListRefPointer, ok := conListRef.ElementValue.(*types.ValuePointer) //
	require.Equal(t, true, ok, "Expected the result to be a ValuePointer, got %v", typename(conListRef.ElementValue))
	conListReferent, ok := conListRefPointer.Referent.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer, got %v", typename(conListRefPointer.Referent))
	require.Equal(t, "ContainerType", conListReferent.TypeName())
	require.Equal(t, 9000, conListRef.MaxSize)
}

func TestListOfOverlays(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	overlayListRep, err := container.GetField("OverlayList")
	require.NoError(t, err)
	require.Equal(t, "[]AliasedPrimitive", overlayListRep.TypeName())
	overlayList, ok := overlayListRep.(*types.ValueList)
	require.Equal(t, true, ok, "Expected a ValueList, got %v", typename(overlayListRep))
	require.Equal(t, 11, overlayList.MaxSize)
	require.Equal(t, "AliasedPrimitive", overlayList.ElementValue.TypeName())
	overlay, ok := overlayList.ElementValue.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected a ValueOverly, got %v", typename(overlayList.ElementValue))
	require.Equal(t, "uint64", overlay.Underlying.TypeName())
	underlying, ok := overlay.Underlying.(*types.ValueUint)
	require.Equal(t, true, ok, "Expected a ValueUint, got %v", typename(overlay.Underlying))
	require.Equal(t, types.UintSize(64), underlying.Size)

	overlayListRefRep, err := container.GetField("OverlayListRef")
	require.NoError(t, err)
	require.Equal(t, "[]*AliasedPrimitive", overlayListRefRep.TypeName())
	overlayRefList, ok := overlayListRefRep.(*types.ValueList)
	require.Equal(t, true, ok, "Expected a ValueList, got %v", typename(overlayListRep))
	require.Equal(t, 58, overlayRefList.MaxSize)
	require.Equal(t, "*AliasedPrimitive", overlayRefList.ElementValue.TypeName())
	overlayPointer, ok := overlayRefList.ElementValue.(*types.ValuePointer)
	require.Equal(t, true, ok, "Expected a ValuePointer, got %v", typename(overlayRefList.ElementValue))
	require.Equal(t, "AliasedPrimitive", overlayPointer.Referent.TypeName())
	overlayRef, ok := overlayPointer.Referent.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected a ValueOverlay, got %v", typename(overlayPointer.Referent))
	require.Equal(t, "uint64", overlayRef.Underlying.TypeName())
	underlyingRef, ok := overlay.Underlying.(*types.ValueUint)
	require.Equal(t, true, ok, "Expected a ValueUint, got %v", typename(overlayRef.Underlying))
	require.Equal(t, types.UintSize(64), underlyingRef.Size)
}

func TestVectorOfOverlays(t *testing.T) {
	typeName := noImports
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	overlayVectorRep, err := container.GetField("OverlayVector")
	require.NoError(t, err)
	require.Equal(t, "[]AliasedPrimitive", overlayVectorRep.TypeName())
	overlayVector, ok := overlayVectorRep.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected a ValueList, got %v", typename(overlayVectorRep))
	require.Equal(t, 23, overlayVector.Size)
	require.Equal(t, "AliasedPrimitive", overlayVector.ElementValue.TypeName())
	overlay, ok := overlayVector.ElementValue.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected a ValueOverly, got %v", typename(overlayVector.ElementValue))
	require.Equal(t, "uint64", overlay.Underlying.TypeName())
	underlying, ok := overlay.Underlying.(*types.ValueUint)
	require.Equal(t, true, ok, "Expected a ValueUint, got %v", typename(overlay.Underlying))
	require.Equal(t, types.UintSize(64), underlying.Size)

	overlayVectorRefRep, err := container.GetField("OverlayVectorRef")
	require.NoError(t, err)
	require.Equal(t, "[]*AliasedPrimitive", overlayVectorRefRep.TypeName())
	overlayRefVector, ok := overlayVectorRefRep.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected a ValueVector, got %v", typename(overlayVectorRep))
	require.Equal(t, 13, overlayRefVector.Size)
	require.Equal(t, "*AliasedPrimitive", overlayRefVector.ElementValue.TypeName())
	overlayPointer, ok := overlayRefVector.ElementValue.(*types.ValuePointer)
	require.Equal(t, true, ok, "Expected a ValuePointer, got %v", typename(overlayRefVector.ElementValue))
	require.Equal(t, "AliasedPrimitive", overlayPointer.Referent.TypeName())
	overlayRef, ok := overlayPointer.Referent.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected a ValueOverlay, got %v", typename(overlayPointer.Referent))
	require.Equal(t, "uint64", overlayRef.Underlying.TypeName())
	underlyingRef, ok := overlay.Underlying.(*types.ValueUint)
	require.Equal(t, true, ok, "Expected a ValueUint, got %v", typename(overlayRef.Underlying))
	require.Equal(t, types.UintSize(64), underlyingRef.Size)
}

func TestBitlist(t *testing.T) {
	typeName := "TestBitlist"
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueContainer)
	require.Equal(t, true, ok, "Expected \"TestBitlist\" to be type ValueContainer, got %v", typename(val))

	overlayValRep, err := container.GetField("AggregationBits")
	require.NoError(t, err)
	overlay, ok := overlayValRep.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected the result to be a ValueOverlay type, got %v", typename(overlayValRep))
	require.Equal(t, "Bitlist", overlay.TypeName())
	require.Equal(t, "[]byte", overlay.Underlying.TypeName())
	underlying, ok := overlay.Underlying.(*types.ValueList)
	require.Equal(t, true, ok, "Expected the result to be a ValueList type, got %v", typename(overlayValRep))
	require.Equal(t, 2048, underlying.MaxSize)
	require.Equal(t, "byte", underlying.ElementValue.TypeName())
	_, ok = underlying.ElementValue.(*types.ValueByte)
	require.Equal(t, true, ok, "Expected the result to be a ValueByte type, got %v", typename(underlying.ElementValue))

	overlayVecValRep, err := container.GetField("JustificationBits")
	require.NoError(t, err)
	overlayVec, ok := overlayVecValRep.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected the result to be a ValueOverlay type, got %v", typename(overlayVec))
	require.Equal(t, "Bitvector4", overlayVec.TypeName())
	require.Equal(t, "[]byte", overlay.Underlying.TypeName())
	underlyingVec, ok := overlayVec.Underlying.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the result to be a ValueVector type, got %v", typename(overlayVecValRep))
	require.Equal(t, 1, underlyingVec.Size)
	require.Equal(t, "byte", underlyingVec.ElementValue.TypeName())
	_, ok = underlyingVec.ElementValue.(*types.ValueByte)
	require.Equal(t, true, ok, "Expected the result to be a ValueByte type, got %v", typename(underlyingVec.ElementValue))
}

func TestFixedSizeArray(t *testing.T) {
	typeName := "FixedSizeArray"
	pp, err := NewPackageParser(packageName, []string{typeName})
	require.NoError(t, err)

	td := pp.results[0]
	val, err := ParseTypeDef(td)
	require.NoError(t, err)
	require.Equal(t, typeName, val.TypeName())
	container, ok := val.(*types.ValueOverlay)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(container))

	underlying := container.Underlying
	c2, ok := underlying.(*types.ValueVector)
	require.Equal(t, true, ok, "Expected the result to be a ValueContainer type, got %v", typename(c2))
}

func typename(v interface{}) string {
	ty := reflect.TypeOf(v)
	if ty.Kind() == reflect.Ptr {
		return "*" + ty.Elem().Name()
	} else {
		return ty.Name()
	}
}
