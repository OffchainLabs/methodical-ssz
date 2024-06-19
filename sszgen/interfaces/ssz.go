package interfaces

import (
	"go/types"

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

func init() {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedTypes}, "github.com/prysmaticlabs/fastssz")
	if err != nil {
		panic(err)
	}
	if len(pkgs) == 0 {
		panic("missing package, add github.com/prysmaticlabs/fastssz to your go.mod")
	}
	SszMarshaler = pkgs[0].Types.Scope().Lookup("Marshaler").Type().Underlying().(*types.Interface)
	SszUnmarshaler = pkgs[0].Types.Scope().Lookup("Unmarshaler").Type().Underlying().(*types.Interface)
	SszFullHasher = pkgs[0].Types.Scope().Lookup("HashRoot").Type().Underlying().(*types.Interface)

	for i := 0; i < SszFullHasher.NumMethods(); i++ {
		method := SszFullHasher.Method(i)
		if method.Name() == "HashTreeRoot" {
			SszLightHasher = types.NewInterfaceType([]*types.Func{method}, nil)
			break
		}
	}
}
