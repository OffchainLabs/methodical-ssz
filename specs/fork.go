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
	//EIP4844     Fork = "eip4844"
)

// var ForkOrder = []Fork{Phase0, Altair, Bellatrix, Capella, EIP4844}
var ForkOrder = []Fork{Phase0, Altair, Bellatrix, Capella}

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
		/*
			case string(EIP4844):
				return EIP4844
		*/
	default:
		return ForkUnknown
	}
}

func forkIndex(f Fork) (int, error) {
	for i := 0; i < len(ForkOrder); i++ {
		if ForkOrder[i] == f {
			return i, nil
		}
	}
	return 0, ErrUnknownFork
}
