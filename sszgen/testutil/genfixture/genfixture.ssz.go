package genfixture

import (
	binary "encoding/binary"
	"fmt"
	go_bitfield "github.com/OffchainLabs/go-bitfield"
	ssz "github.com/OffchainLabs/methodical-ssz/ssz"
)

func (c *Bits) SizeSSZ() int {
	size := 7
	size += len(c.L)
	return size
}

func (c *Bits) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *Bits) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error
	offset := 7

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

	// Field 2: L
	dst = ssz.WriteOffset(dst, offset)
	offset += len(c.L)

	// Field 2: L
	if err = ssz.ValidateBitlist(c.L, 2048); err != nil {
		return nil, fmt.Errorf("L: %w", err)
	}
	dst = append(dst, c.L...)
	return dst, err
}

func (c *Bits) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size < 7 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:1] // c.V4
	sszSlice1 := buf[1:3] // c.V16

	sszVarOffset2 := ssz.ReadOffset(buf[3:7]) // c.L
	if sszVarOffset2 != 7 {
		return ssz.ErrInvalidVariableOffset
	}
	if sszVarOffset2 > size {
		return ssz.ErrOffset
	}
	sszSlice2 := buf[sszVarOffset2:] // c.L

	// Field 0: V4
	if err = ssz.ValidateBitvector(sszSlice0, 4); err != nil {
		return fmt.Errorf("V4: %w", err)
	}
	c.V4 = make([]byte, 0, 1)
	c.V4 = append(c.V4, go_bitfield.Bitvector4(sszSlice0)...)

	// Field 1: V16
	c.V16 = make([]byte, 0, 2)
	c.V16 = append(c.V16, go_bitfield.Bitvector16(sszSlice1)...)

	// Field 2: L
	if err = ssz.ValidateBitlist(sszSlice2, 2048); err != nil {
		return fmt.Errorf("L: %w", err)
	}
	c.L = append([]byte{}, go_bitfield.Bitlist(sszSlice2)...)
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
	// Field 2: L
	if len(c.L) == 0 {
		return ssz.ErrEmptyBitlist
	}
	hh.PutBitlist(c.L, 2048)
	hh.Merkleize(indx)
	return nil
}

func (c *Limbs) SizeSSZ() int {
	size := 32

	return size
}

func (c *Limbs) MarshalSSZ() ([]byte, error) {
	buf := make([]byte, c.SizeSSZ())
	return c.MarshalSSZTo(buf[:0])
}

func (c *Limbs) MarshalSSZTo(dst []byte) ([]byte, error) {
	var err error

	// Field 0: A
	if len(c.A) != 4 {
		return nil, ssz.ErrBytesLength
	}
	for _, o := range c.A {
		dst = binary.LittleEndian.AppendUint64(dst, o)
	}

	return dst, err
}

func (c *Limbs) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 32 {
		return ssz.ErrSize
	}

	sszSlice0 := buf[0:32] // c.A

	// Field 0: A
	{

		for i := 0; i < 4; i++ {
			var tmp uint64

			tmpSlice := sszSlice0[i*8 : (1+i)*8]
			tmp = binary.LittleEndian.Uint64(tmpSlice)
			c.A[i] = tmp
		}
	}
	return err
}

func (c *Limbs) HashTreeRoot() ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := c.HashTreeRootWith(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)
	return root, err
}

func (c *Limbs) HashTreeRootWith(hh *ssz.Hasher) (err error) {
	indx := hh.Index()
	// Field 0: A
	{
		if len(c.A) != 4 {
			return ssz.ErrVectorLength
		}
		subIndx := hh.Index()
		for _, o := range c.A {
			hh.AppendUint64(o)
		}
		hh.Merkleize(subIndx)
	}
	hh.Merkleize(indx)
	return nil
}
