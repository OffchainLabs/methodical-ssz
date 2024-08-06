package specs

import "github.com/pkg/errors"

type Fork string

var ErrUnknownFork = errors.New("unknown fork name")

func (f *Fork) UnmarshalText(t []byte) error {
	s := string(t)
	sf := stringToFork(s)
	if sf == ForkUnknown {
		return errors.Wrap(ErrUnknownFork, s)
	}
	*f = sf
	return nil
}

var (
	ForkUnknown Fork = ""
	Phase0      Fork = "phase0"
	Altair      Fork = "altair"
	Bellatrix   Fork = "bellatrix"
	Capella     Fork = "capella"
	Deneb       Fork = "deneb"
	Electra     Fork = "electra"
	Fulu        Fork = "fulu"
	Gloas       Fork = "gloas"
)

// ForkOrder is the canonical, chronological ordering of forks. RelationsAtFork
// walks it backwards to resolve type inheritance, so new forks must be appended
// in activation order.
var ForkOrder = []Fork{Phase0, Altair, Bellatrix, Capella, Deneb, Electra, Fulu, Gloas}

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
	case string(Deneb):
		return Deneb
	case string(Electra):
		return Electra
	case string(Fulu):
		return Fulu
	case string(Gloas):
		return Gloas
	default:
		return ForkUnknown
	}
}

func ForkIndex(f Fork) (int, error) {
	for i := 0; i < len(ForkOrder); i++ {
		if ForkOrder[i] == f {
			return i, nil
		}
	}
	return 0, ErrUnknownFork
}
