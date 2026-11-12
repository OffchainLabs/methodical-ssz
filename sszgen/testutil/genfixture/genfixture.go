// Package genfixture holds compiled generator output for shapes where formal
// verification found generated code deviating from the SSZ spec. The
// generated file genfixture.ssz.go is checked in and pinned by
// sszgen.TestGenFixture (regenerate with `go test ./sszgen -run
// TestGenFixture -update`); the tests in this package execute it.
package genfixture

import bitfield "github.com/OffchainLabs/go-bitfield"

// Bits covers bitfields: a padded Bitvector[4], an unpadded Bitvector[16]
// and a Bitlist[2048].
type Bits struct {
	V4  bitfield.Bitvector4  `ssz-size:"1"`
	V16 bitfield.Bitvector16 `ssz-size:"2"`
	L   bitfield.Bitlist     `ssz-max:"2048"`
}

// Limbs covers a Go array of uint64 (Vector[uint64, 4] as [4]uint64).
type Limbs struct {
	A [4]uint64
}

// ListU64 covers a fixed field followed by a bounded uint64 list.
type ListU64 struct {
	A uint64
	L []uint64 `ssz-max:"4"`
}

// Inner is a variable-size container.
type Inner struct {
	B []byte `ssz-max:"8"`
}

// InnerPair covers a Go array of variable-size elements (Vector[Inner, 2]).
type InnerPair struct {
	P [2]*Inner
}
