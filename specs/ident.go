package specs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Preset string

var (
	PresetUnknown Preset = ""
	Minimal       Preset = "minimal"
	Mainnet       Preset = "mainnet"
)

var ErrUnknownPreset = errors.New("unknown preset name")

func (p *Preset) UnmarshalText(t []byte) error {
	s := string(t)
	sp := stringToPreset(s)
	if sp == PresetUnknown {
		return errors.Wrap(ErrUnknownPreset, s)
	}
	*p = sp
	return nil
}

func stringToPreset(s string) Preset {
	switch s {
	case string(Minimal):
		return Minimal
	case string(Mainnet):
		return Mainnet
	default:
		return PresetUnknown
	}
}

type TestIdent struct {
	Preset Preset `json:"preset"`
	Fork   Fork   `json:"fork"`
	Name   string `json:"name"`
	Offset int    `json:"offset"`
}

func (ti TestIdent) String() string {
	return fmt.Sprintf("%s:%s:%s:%d", ti.Preset, ti.Fork, ti.Name, ti.Offset)
}

var layout = struct {
	testDir   int
	preset    int
	fork      int
	sszStatic int
	typeName  int
	sszRandom int
	caseNum   int
	fileName  int
}{
	testDir:   0,
	preset:    1,
	fork:      2,
	sszStatic: 3,
	typeName:  4,
	sszRandom: 5,
	caseNum:   6,
	fileName:  7,
}

func (ti TestIdent) Match(other TestIdent) bool {
	if other.Preset == PresetUnknown || other.Fork == ForkUnknown || other.Name == "" {
		return false
	}
	if ti.Preset != PresetUnknown && ti.Preset != other.Preset {
		return false
	}
	if ti.Fork != ForkUnknown && ti.Fork != other.Fork {
		return false
	}
	if ti.Name != "" && ti.Name != other.Name {
		return false
	}
	return true
}

var ErrPathParse = errors.New("spectest path not in expected format, could not parse identifiers")

var caseOffset = len("case_")

func ParsePath(p string) (TestIdent, string, error) {
	parts := strings.Split(p, "/")
	if len(parts) <= layout.fileName || parts[layout.testDir] != "tests" || parts[layout.sszStatic] != "ssz_static" || parts[layout.sszRandom] != "ssz_random" {
		return TestIdent{}, "", nil
	}
	var offset int
	if len(parts[layout.caseNum]) > caseOffset {
		a, err := strconv.Atoi(parts[layout.caseNum][caseOffset:])
		if err != nil {
			return TestIdent{}, "", errors.Wrapf(err, "problem parsing case number from path %s", p)
		}
		offset = a
	}
	return TestIdent{
		Preset: stringToPreset(parts[layout.preset]),
		Fork:   stringToFork(parts[layout.fork]),
		Name:   parts[layout.typeName],
		Offset: offset,
	}, parts[layout.fileName], nil
}
