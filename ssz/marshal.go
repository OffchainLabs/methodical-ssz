package ssz

import (
	"encoding/binary"
)

// Sizer is the interface implemented by types that can report their own ssz
// encoded size. It is the subset of Marshaler used by size delegation.
type Sizer interface {
	SizeSSZ() int
}

// Marshaler is the interface implemented by types that can marshal themselves into valid SZZ.
type Marshaler interface {
	MarshalSSZTo(dst []byte) ([]byte, error)
	MarshalSSZ() ([]byte, error)
	Sizer
}

// MarshalSSZ marshals an object
func MarshalSSZ(m Marshaler) ([]byte, error) {
	buf := make([]byte, m.SizeSSZ())
	return m.MarshalSSZTo(buf[:0])
}

// WriteOffset writes an offset to dst
func WriteOffset(dst []byte, i int) []byte {
	return binary.LittleEndian.AppendUint32(dst, uint32(i))
}
