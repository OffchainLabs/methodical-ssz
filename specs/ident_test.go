package specs

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestParsePath(t *testing.T) {
	cases := []struct{
		name string
		path string
		err error
		ident TestIdent
		match bool
	}{
		{
			name: "mainnet capella",
			match: true,
			path: "tests/mainnet/capella/ssz_static/LightClientOptimisticUpdate/ssz_random/case_0",
			ident: TestIdent{
				Preset:   &Mainnet,
				Fork:     &Capella,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			other, err := ParsePath(c.path)
			if c.err == nil {
				require.NoError(t, err)
			}
			require.Equal(t, c.match, c.ident.Match(other))
		})
	}
}