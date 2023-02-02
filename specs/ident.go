package specs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Fork string

var (
	ForkUnknown Fork = ""
	Phase0      Fork = "phase0"
	Altair      Fork = "altair"
	Bellatrix   Fork = "bellatrix"
	Capella     Fork = "capella"
	EIP4844     Fork = "eip4844"
)

func stringToFork(s string) Fork {
	switch s {
	case string(Phase0):
		return Phase0
	case string(Altair):
		return Altair
	case string(Bellatrix):
		return Bellatrix
	case string(Capella):
		return Capella
	case string(EIP4844):
		return EIP4844
	default:
		return ForkUnknown
	}
}

type Preset string

var (
	PresetUnknown Preset = ""
	Minimal       Preset = "minimal"
	Mainnet       Preset = "mainnet"
)

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
	Preset   Preset
	Fork     Fork
	TypeName string
	Offset   int
}

func (ti TestIdent) String() string {
	return fmt.Sprintf("%s:%s:%s:%d", ti.Preset, ti.Fork, ti.TypeName, ti.Offset)
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
	if other.Preset == PresetUnknown || other.Fork == ForkUnknown || other.TypeName == "" {
		return false
	}
	if ti.Preset != PresetUnknown && ti.Preset != other.Preset {
		return false
	}
	if ti.Fork != ForkUnknown && ti.Fork != other.Fork {
		return false
	}
	if ti.TypeName != "" && ti.TypeName != other.TypeName {
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
		Preset:   stringToPreset(parts[layout.preset]),
		Fork:     stringToFork(parts[layout.fork]),
		TypeName: parts[layout.typeName],
		Offset:   offset,
	}, parts[layout.fileName], nil
}
