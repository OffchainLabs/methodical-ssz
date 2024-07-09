package interfaces

import (
	"fmt"
	"go/types"

	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"golang.org/x/tools/go/packages"
)

var (
	SszMarshaler   *types.Interface
	SszUnmarshaler *types.Interface
	SszFullHasher  *types.Interface
	SszLightHasher *types.Interface
)

func NewSSZSupportMap(t types.Type) map[*types.Interface]bool {
	return map[*types.Interface]bool{
		SszMarshaler:   types.Implements(t, SszMarshaler) || types.Implements(types.NewPointer(t), SszMarshaler),
		SszUnmarshaler: types.Implements(t, SszUnmarshaler) || types.Implements(types.NewPointer(t), SszUnmarshaler),
		SszLightHasher: types.Implements(t, SszLightHasher) || types.Implements(types.NewPointer(t), SszLightHasher),
		SszFullHasher:  types.Implements(t, SszFullHasher) || types.Implements(types.NewPointer(t), SszFullHasher),
	}
}

// Hack to make sure the fastssz package is imported and included our go.mod.
// This is needed for the package reflection in the init method below.
var _ = ssz.Marshaler(nil)
var _ = ssz.Unmarshaler(nil)
var _ = ssz.HashRoot(nil)

func loadSsz() (*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedTypes}, "github.com/prysmaticlabs/fastssz")
	if err != nil {
		return nil, errors.Wrap(err, "error from packages.Load for fastssz")
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("missing package, add github.com/prysmaticlabs/fastssz to your go.mod")
	}
	for _, p := range pkgs {
		if p.ID == "github.com/prysmaticlabs/fastssz" {
			return p, nil
		}
	}
	return nil, fmt.Errorf("github.com/prysmaticlabs/fastssz not found in go package index - add to your go.mod")
}

func init() {
	pkg, err := loadSsz()
	if err != nil {
		panic(err)
	}
	SszMarshaler = pkg.Types.Scope().Lookup("Marshaler").Type().Underlying().(*types.Interface)
	SszUnmarshaler = pkg.Types.Scope().Lookup("Unmarshaler").Type().Underlying().(*types.Interface)
	SszFullHasher = pkg.Types.Scope().Lookup("HashRoot").Type().Underlying().(*types.Interface)

	for i := 0; i < SszFullHasher.NumMethods(); i++ {
		method := SszFullHasher.Method(i)
		if method.Name() == "HashTreeRoot" {
			SszLightHasher = types.NewInterfaceType([]*types.Func{method}, nil)
			break
		}
	}
}
