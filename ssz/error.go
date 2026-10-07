package ssz

import (
	"errors"
	"fmt"
)

var (
	ErrBytesLength           = errors.New("bytes array does not have the correct length")
	ErrEmptyBitlist          = errors.New("bitlist is empty")
	ErrInvalidSerialization  = errors.New("invalid serialization for the expected ssz type")
	ErrInvalidVariableOffset = errors.New("invalid ssz encoding. first variable element offset indexes into fixed value data")
	ErrListTooBig            = errors.New("list length is higher than max value")
	ErrOffset                = errors.New("incorrect offset")
	ErrSize                  = errors.New("incorrect size")
	ErrVectorLength          = errors.New("vector does not have the correct length")

	ErrIncorrectByteSize = fmt.Errorf("incorrect byte size")
	ErrIncorrectListSize = fmt.Errorf("incorrect list size")
	ErrRootSizeInvalid   = errors.New("root must be 32 bytes")
)
