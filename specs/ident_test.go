package specs

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
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
			if c.err == nil && err != nil {
				t.Fatalf("unexpected error=%s", err.Error())
			}
			if c.ident.Match(other) != c.match {
				t.Fatalf("unexpected test identifier, want=%t, got=%t", c.match, c.ident.Match(other))
			}
			if fname != c.fname {
				t.Fatalf("unexpected file name for identifier config, want=%s, got=%s", c.fname, fname)
			}
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
				if err != nil {
					t.Fatalf("unexpected error=%s", err.Error())
				}
			} else {
				if !errors.Is(err, c.err) {
					t.Fatalf("did not get expected error, want=%s, got=%s", c.err.Error(), err.Error())
				}
			}
			if c.fork != nil && ti.Fork != *c.fork {
				t.Fatalf("wanted fork=%s, got=%s", *c.fork, ti.Fork)
			}
			if c.preset != nil && ti.Preset != *c.preset {
				t.Fatalf("wanted preset=%s, got=%s", *c.preset, ti.Preset)
			}
		})
	}
}
