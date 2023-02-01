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
	Phase0 Fork = "phase0"
	Altair Fork = "altair"
	Bellatrix Fork = "bellatrix"
	Capella Fork = "capella"
	EIP4844 Fork = "eip4844"
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
	Minimal Preset = "minimal"
	Mainnet Preset = "mainnet"
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
	Preset *Preset
	Fork *Fork
	TypeName *string
	Offset *int
}

func (ti TestIdent) String() string {
	preset := ""
	fork := ""
	typeName := ""
	offset := 0
	if ti.Preset != nil {
		preset = string(*ti.Preset)
	}
	if ti.Fork != nil {
		fork = string(*ti.Fork)
	}
	if ti.TypeName != nil {
		typeName = *ti.TypeName
	}
	if ti.Offset != nil {
		offset = *ti.Offset
	}
	return fmt.Sprintf("%s:%s:%s:%d", preset, fork, typeName, offset)
}

var layout = struct {
	testDir int
	preset int
	fork int
	sszStatic int
	typeName int
	sszRandom int
	caseNum int
}{
	testDir: 0,
	preset: 1,
	fork: 2,
	sszStatic: 3,
	typeName: 4,
	sszRandom: 5,
	caseNum: 6,
}

func (ti TestIdent) Match(other TestIdent) bool {
	if other.Preset == nil || other.Fork == nil || other.TypeName == nil || other.Offset == nil {
		return false
	}
	if ti.Preset != nil && *ti.Preset != *other.Preset {
		return false
	}
	if ti.Fork != nil && *ti.Fork != *other.Fork {
		return false
	}
	if ti.TypeName != nil && *ti.TypeName != *other.TypeName {
		return false
	}
	if ti.Offset != nil && *ti.Offset != *other.Offset {
		return false
	}
	return true
}

var ErrPathParse = errors.New("spectest path not in expected format, could not parse identifiers")

var caseOffset = len("case_")
func ParsePath(p string) (TestIdent, error) {
	parts := strings.Split(p, "/")
	if len(parts) <= layout.caseNum {
		return TestIdent{}, nil
	}
	if parts[0] != "tests" {
		return TestIdent{}, ErrPathParse
	}
	preset := stringToPreset(parts[layout.preset])
	fork := stringToFork(parts[layout.fork])
	name := parts[layout.typeName]
	var offset *int = nil
	if len(parts[layout.caseNum]) > caseOffset {
		a, err := strconv.Atoi(parts[layout.caseNum][caseOffset:])
		if err != nil {
			return TestIdent{}, err
		}
		offset = &a
	}
	return TestIdent{
		Preset: &preset,
		Fork: &fork,
		TypeName: &name,
		Offset: offset,
	}, nil
}