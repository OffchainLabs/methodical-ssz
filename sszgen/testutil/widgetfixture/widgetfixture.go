// Package widgetfixture is a second compiled fixture package (sibling to
// delegatefixture), used to exercise multi-package spec test generation: one
// generated test package importing types from both.
package widgetfixture

import "github.com/OffchainLabs/methodical-ssz/ssz"

// Gadget is a minimal fixed-size type with its own SSZ methodset.
type Gadget [4]byte

func (g *Gadget) SizeSSZ() int { return 4 }

func (g *Gadget) MarshalSSZTo(dst []byte) ([]byte, error) {
	return append(dst, g[:]...), nil
}

func (g *Gadget) MarshalSSZ() ([]byte, error) {
	return g.MarshalSSZTo(make([]byte, 0, 4))
}

func (g *Gadget) UnmarshalSSZ(buf []byte) error {
	if len(buf) != 4 {
		return ssz.ErrSize
	}
	copy(g[:], buf)
	return nil
}

func (g *Gadget) HashTreeRoot() ([32]byte, error) {
	var root [32]byte
	copy(root[:], g[:])
	return root, nil
}
