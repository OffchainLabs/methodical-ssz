package genfixture

import (
	"fmt"
	go_bitfield "github.com/OffchainLabs/go-bitfield"
	ssz "github.com/OffchainLabs/methodical-ssz/ssz"
)

func (c *Bits) SizeSSZ() int {
	size := 3

	return size
}

func (c *Bits) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *Bits) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: V4
	if err = ssz.ValidateBitvector(c.V4, 4); err != nil {
		return nil, fmt.Errorf("V4: %w", err)
	}
	if len([]byte(c.V4)) != 1 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, []byte(c.V4)...)

	// Field 1: V16
	if len([]byte(c.V16)) != 2 {
		return nil, ssz.ErrBytesLength
	}
	dst = append(dst, []byte(c.V16)...)

	return dst, err
}

func (c *Bits) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 3 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:1] // c.V4
	sszSlice1 := buf[1:3] // c.V16

	// Field 0: V4
	if err = ssz.ValidateBitvector(sszSlice0, 4); err != nil {
		return fmt.Errorf("V4: %w", err)
	}
	c.V4 = make([]byte, 0, 1)
	c.V4 = append(c.V4, go_bitfield.Bitvector4(sszSlice0)...)

	// Field 1: V16
	c.V16 = make([]byte, 0, 2)
	c.V16 = append(c.V16, go_bitfield.Bitvector16(sszSlice1)...)
	return err
}

func (c *Bits) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Bits) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: V4
	if len([]byte(c.V4)) != 1 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes([]byte(c.V4))
	// Field 1: V16
	if len([]byte(c.V16)) != 2 {
		return ssz.ErrBytesLength
	}
	hh.PutBytes([]byte(c.V16))
	hh.Merkleize(indx)
	return nil
}
