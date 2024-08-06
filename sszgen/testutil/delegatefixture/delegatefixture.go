// Package delegatefixture provides real types implementing the SSZ runtime
// interfaces, used by the end-to-end delegation tests (see test-matrix.md
// Phase 3). The package is compiled as part of the module so the method
// signatures — including HashTreeRootWith's *ssz.Hasher parameter, which a
// virtual-file fixture cannot import — are verified by the compiler, and so a
// future round-trip harness can execute against it.
package delegatefixture

import (
	"encoding/binary"

	"github.com/OffchainLabs/methodical-ssz/ssz"
)

// DelegateContainer is the generation target: every field delegates to its
// type's own SSZ methods, in scalar and element positions, fixed and variable.
type DelegateContainer struct {
	Full  *Wide
	Val   Light
	Items []*Wide `ssz-max:"4"`
	B     *Blob   `ssz-max:"256"`
	// NOTE: the tag re-applies wholesale through the pointer/named
	// indirection, so the outer list and the inner blob share the 256 limit —
	// deliberately matching blobLimit so the type is self-consistent.
	Blobs []*Blob `ssz-max:"256"`
}

// Wide is a fixed-size non-container type with pointer-receiver methods
// implementing the complete SSZ method set, including the full hasher
// (HashTreeRootWith) — the delegate branch no other fixture exercises.
type Wide [8]byte

func (w *Wide) SizeSSZ() int { return 8 }

func (w *Wide) MarshalSSZTo(dst []byte) ([]byte, error) {
	return append(dst, w[:]...), nil
}

func (w *Wide) MarshalSSZ() ([]byte, error) {
	return w.MarshalSSZTo(make([]byte, 0, 8))
}

func (w *Wide) UnmarshalSSZ(buf []byte) error {
	if len(buf) != 8 {
		return ssz.ErrSize
	}
	copy(w[:], buf)
	return nil
}

func (w *Wide) HashTreeRoot() ([32]byte, error) {
	var root [32]byte
	copy(root[:], w[:])
	return root, nil
}

func (w *Wide) HashTreeRootWith(hh *ssz.Hasher) error {
	hh.PutBytes(w[:])
	return nil
}

// Light is a fixed-size type with value-receiver methods (except UnmarshalSSZ,
// which must mutate) and only the light hasher — a value field that delegates
// without a pointer wrapper.
type Light uint64

func (l Light) SizeSSZ() int { return 8 }

func (l Light) MarshalSSZTo(dst []byte) ([]byte, error) {
	return binary.LittleEndian.AppendUint64(dst, uint64(l)), nil
}

func (l Light) MarshalSSZ() ([]byte, error) {
	return l.MarshalSSZTo(make([]byte, 0, 8))
}

func (l *Light) UnmarshalSSZ(buf []byte) error {
	if len(buf) != 8 {
		return ssz.ErrSize
	}
	*l = Light(binary.LittleEndian.Uint64(buf))
	return nil
}

func (l Light) HashTreeRoot() ([32]byte, error) {
	var root [32]byte
	binary.LittleEndian.PutUint64(root[:8], uint64(l))
	return root, nil
}

// Blob is a variable-size delegated type: its runtime SizeSSZ participates in
// the generated size/offset bookkeeping, in scalar and element positions.
type Blob []byte

const blobLimit = 256

func (b *Blob) SizeSSZ() int { return len(*b) }

func (b *Blob) MarshalSSZTo(dst []byte) ([]byte, error) {
	if len(*b) > blobLimit {
		return nil, ssz.ErrListTooBig
	}
	return append(dst, *b...), nil
}

func (b *Blob) MarshalSSZ() ([]byte, error) {
	return b.MarshalSSZTo(make([]byte, 0, len(*b)))
}

func (b *Blob) UnmarshalSSZ(buf []byte) error {
	if len(buf) > blobLimit {
		return ssz.ErrListTooBig
	}
	*b = append((*b)[:0], buf...)
	return nil
}

func (b *Blob) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	indx := hh.Index()
	hh.AppendBytes32(*b)
	hh.MerkleizeWithMixin(indx, uint64(len(*b)), (blobLimit+31)/32)
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}
