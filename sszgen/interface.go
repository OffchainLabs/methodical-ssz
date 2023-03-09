package sszgen

import (
	"go/types"

	"golang.org/x/tools/go/packages"
)

var (
	fastsszMarshaler   *types.Interface
	fastsszUnmarshaler *types.Interface
	fastsszFullHasher  *types.Interface
	fastsszLightHasher *types.Interface
)

func init() {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedTypes}, "github.com/prysmaticlabs/fastssz")
	if err != nil {
		panic(err)
	}
	if len(pkgs) == 0 {
		panic("missing package, add github.com/prysmaticlabs/fastssz to your go.mod")
	}
	fastsszMarshaler = pkgs[0].Types.Scope().Lookup("Marshaler").Type().Underlying().(*types.Interface)
	fastsszUnmarshaler = pkgs[0].Types.Scope().Lookup("Unmarshaler").Type().Underlying().(*types.Interface)
	fastsszFullHasher = pkgs[0].Types.Scope().Lookup("HashRoot").Type().Underlying().(*types.Interface)

	for i := 0; i < fastsszFullHasher.NumMethods(); i++ {
		method := fastsszFullHasher.Method(i)
		if method.Name() == "HashTreeRoot" {
			fastsszLightHasher = types.NewInterfaceType([]*types.Func{method}, nil)
			break
		}
	}
}
