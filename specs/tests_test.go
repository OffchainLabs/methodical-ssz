package specs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestDecodeRootFile(t *testing.T) {
	e, err := hexutil.Decode("0x44de62c118d7951f5b6d9a03444e54aff47d02ff57add2a4eb2a198b3e83ae35")
	expected := [32]byte{}
	copy(expected[:], e)
	require.NoError(t, err)
	f := []byte(`{root: '0x44de62c118d7951f5b6d9a03444e54aff47d02ff57add2a4eb2a198b3e83ae35'}`)
	r, err := decodeRootFile(f)
	require.NoError(t, err)
	require.Equal(t, expected, r)
}
