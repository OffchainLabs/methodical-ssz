// Package genfixture holds compiled generator output for shapes where formal
// verification found generated code deviating from the SSZ spec. The
// generated file genfixture.ssz.go is checked in and pinned by
// sszgen.TestGenFixture (regenerate with `go test ./sszgen -run
// TestGenFixture -update`); the tests in this package execute it.
package genfixture

import bitfield "github.com/OffchainLabs/go-bitfield"

// Bits covers bitfields: a padded Bitvector[4] and an unpadded Bitvector[16].
type Bits struct {
	V4  bitfield.Bitvector4  `ssz-size:"1"`
	V16 bitfield.Bitvector16 `ssz-size:"2"`
}
