// Package genfixture compiles sszgen output so that tests can run it.
// sszgen.TestGenFixture pins genfixture.ssz.go.
//
// To regenerate it, run:
// `go test ./sszgen -run TestGenFixture -update`.
package genfixture

import bitfield "github.com/OffchainLabs/go-bitfield"

type Bits struct {
	V4  bitfield.Bitvector4  `ssz-size:"1"`
	V16 bitfield.Bitvector16 `ssz-size:"2"`
}

type Bits2 struct {
	V2 bitfield.Bitvector2 `ssz-size:"1"`
}
