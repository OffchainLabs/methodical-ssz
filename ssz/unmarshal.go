package ssz

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// Unmarshaler is the interface implemented by types that can unmarshal a SSZ description of themselves
type Unmarshaler interface {
	UnmarshalSSZ(buf []byte) error
}

const bytesPerLengthOffset = 4

// ValidateProgressiveBitlist validates the encoding of a progressive bitlist,
// which has no length limit: it must be non-empty and carry its delimiter bit
// in a non-zero trailing byte.
func ValidateProgressiveBitlist(buf []byte) error {
	if len(buf) == 0 {
		return fmt.Errorf("bitlist empty, it does not have length bit")
	}
	if buf[len(buf)-1] == 0 {
		return fmt.Errorf("trailing byte is zero")
	}
	return nil
}

// ValidateBitlist validates that the bitlist is correct
func ValidateBitlist(buf []byte, bitLimit uint64) error {
	byteLen := len(buf)
	if byteLen == 0 {
		return fmt.Errorf("bitlist empty, it does not have length bit")
	}
	// Maximum possible bytes in a bitlist with provided bitlimit.
	maxBytes := (bitLimit >> 3) + 1
	if byteLen > int(maxBytes) {
		return fmt.Errorf("unexpected number of bytes, got %d but found %d", byteLen, maxBytes)
	}

	// The most significant bit is present in the last byte in the array.
	last := buf[byteLen-1]
	if last == 0 {
		return fmt.Errorf("trailing byte is zero")
	}

	// Determine the position of the most significant bit.
	msb := bits.Len8(last)

	// The absolute position of the most significant bit will be the number of
	// bits in the preceding bytes plus the position of the most significant
	// bit. Subtract this value by 1 to determine the length of the bitlist.
	numOfBits := uint64(8*(byteLen-1) + msb - 1)

	if numOfBits > bitLimit {
		return fmt.Errorf("too many bits")
	}
	return nil
}

// ReadOffset reads an offset from buf
func ReadOffset(buf []byte) uint64 {
	return uint64(binary.LittleEndian.Uint32(buf))
}
