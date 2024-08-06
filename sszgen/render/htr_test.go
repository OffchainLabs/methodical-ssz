package render

import (
	"fmt"
	"testing"
)

// encodeActiveFields packs a progressive container's active-fields bitvector
// per the spec's pack_bits: bit i is bit i%8 of byte i/8.
func TestEncodeActiveFields(t *testing.T) {
	cases := []struct {
		af       []bool
		expected string
	}{
		{[]bool{true, true, true}, "[]byte{0b00000111}"},
		{[]bool{true, false, true}, "[]byte{0b00000101}"},
		{[]bool{true, true, true, true, true, true, true, true}, "[]byte{0b11111111}"},
		{[]bool{true, true, true, true, true, true, true, true, true}, "[]byte{0b11111111, 0b00000001}"},
		{[]bool{true, false, false, false, false, false, false, false, true}, "[]byte{0b00000001, 0b00000001}"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d_bits", len(c.af)), func(t *testing.T) {
			if got := encodeActiveFields(c.af); got != c.expected {
				t.Fatalf("want %s, got %s", c.expected, got)
			}
		})
	}
}
