package specs

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestParsePath(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		err   error
		ident TestIdent
		fname string
		match bool
	}{
		{
			name:  "mainnet capella",
			match: true,
			path:  "tests/mainnet/capella/ssz_static/LightClientOptimisticUpdate/ssz_random/case_0/roots.yaml",
			ident: TestIdent{
				Preset: Mainnet,
				Fork:   Capella,
			},
			fname: "roots.yaml",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			other, fname, err := ParsePath(c.path)
			if c.err == nil {
				require.NoError(t, err)
			}
			require.Equal(t, c.match, c.ident.Match(other))
			require.Equal(t, c.fname, fname)
		})
	}
}

func TestUnmarshalIdentFields(t *testing.T) {
	cases := []struct {
		name      string
		marshaled string
		err       error
		preset    *Preset
		fork      *Fork
	}{
		{
			name:      "unknown fork",
			marshaled: `{"fork": "derp"}`,
			err:       ErrUnknownFork,
		},
		{
			name:      "altair",
			marshaled: fmt.Sprintf(`{"fork": "%s"}`, Altair),
			fork:      &Altair,
		},
		{
			name:      "phase0",
			marshaled: fmt.Sprintf(`{"fork": "%s"}`, Phase0),
			fork:      &Phase0,
		},
		{
			name:      "unknown preset",
			marshaled: `{"preset": "derp"}`,
			err:       ErrUnknownPreset,
		},
		{
			name:      "mainnet preset",
			marshaled: `{"preset": "mainnet"}`,
			preset:    &Mainnet,
		},
		{
			name:      "minimal preset",
			marshaled: `{"preset": "minimal"}`,
			preset:    &Minimal,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ti := &TestIdent{}
			err := json.Unmarshal([]byte(c.marshaled), ti)
			if c.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, c.err)
			}
			if c.fork != nil {
				require.Equal(t, *c.fork, ti.Fork)
			}
			if c.preset != nil {
				require.Equal(t, *c.preset, ti.Preset)
			}
		})
	}
}
