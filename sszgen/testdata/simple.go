package testdata

import github_com_prysmaticlabs_go_bitfield "github.com/prysmaticlabs/go-bitfield"

type NoImports struct {
	state         int
	sizeCache     int
	unknownFields int

	GenesisTime           uint64   `protobuf:"varint,1001,opt,name=genesis_time,json=genesisTime,proto3" json:"genesis_time,omitempty"`
	GenesisValidatorsRoot []byte   `protobuf:"bytes,1002,opt,name=genesis_validators_root,json=genesisValidatorsRoot,proto3" json:"genesis_validators_root,omitempty" ssz-size:"32"`
	BlockRoots            [][]byte `protobuf:"bytes,2002,rep,name=block_roots,json=blockRoots,proto3" json:"block_roots,omitempty" ssz-size:"8192,32"`
	HistoricalRoots       [][]byte `protobuf:"bytes,2004,rep,name=historical_roots,json=historicalRoots,proto3" json:"historical_roots,omitempty" ssz-max:"16777216" ssz-size:"?,32"`
	MuhPrim               AliasedPrimitive
	ContainerField        ContainerType
	ContainerRefField     *AnotherContainerType
	ContainerList         []ContainerType     `ssz-max:"23"`
	ContainerVector       []ContainerType     `ssz-size:"42"`
	ContainerVectorRef    []*ContainerType    `ssz-size:"17"`
	ContainerListRef      []*ContainerType    `ssz-max:"9000"`
	OverlayList           []AliasedPrimitive  `ssz-max:"11"`
	OverlayListRef        []*AliasedPrimitive `ssz-max:"58"`
	OverlayVector         []AliasedPrimitive  `ssz-size:"23"`
	OverlayVectorRef      []*AliasedPrimitive `ssz-size:"13"`
}

type AliasedPrimitive uint64

type ContainerType struct {
	MuhPrim AliasedPrimitive
}

type AnotherContainerType struct {
	MuhPrim AliasedPrimitive
}

type FixedSizeArray [32]byte

type FixedContainer struct {
	Array FixedSizeArray
}

type TestBitlist struct {
	AggregationBits   github_com_prysmaticlabs_go_bitfield.Bitlist    `protobuf:"bytes,1,opt,name=aggregation_bits,json=aggregationBits,proto3" json:"aggregation_bits,omitempty" cast-type:"github.com/prysmaticlabs/go-bitfield.Bitlist" ssz-max:"2048"`
	JustificationBits github_com_prysmaticlabs_go_bitfield.Bitvector4 `protobuf:"bytes,8001,opt,name=justification_bits,json=justificationBits,proto3" json:"justification_bits,omitempty" cast-type:"github.com/prysmaticlabs/go-bitfield.Bitvector4" ssz-size:"1"`
}
