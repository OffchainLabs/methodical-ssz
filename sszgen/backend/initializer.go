package backend

import (
	"fmt"

	"github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

func initializeValue(rep types.ValRep, pkgName string) string {
	switch vt := rep.(type) {
	case *types.ValuePointer:
		return fmt.Sprintf("new(%s)", fullyQualifiedTypeName(vt.Referent, pkgName))
	default:
		return ""
	}
}
