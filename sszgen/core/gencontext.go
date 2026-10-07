package core

import (
	"github.com/OffchainLabs/methodical-ssz/sszgen/interfaces"
	gentypes "github.com/OffchainLabs/methodical-ssz/sszgen/types"
)

// GenContext carries the per-file services every method set and op handler needs:
// the target package (to decide whether a type reference must be qualified), the
// shared import namer, and the delegation interface set. One GenContext is
// threaded through a whole generation pass so all method sets share the namer
// and the interface identity keys.
type GenContext struct {
	TargetPackage string
	Namer         *ImportNamer
	// Interfaces supplies the delegation-target interfaces for this run. The
	// instance must be the same one the frontend used to build the ValReps'
	// support maps — the pointers are identity keys (see interfaces.Set).
	Interfaces *interfaces.Set
}

// zeroInterfaceSet disables delegation: every lookup field is nil.
var zeroInterfaceSet = &interfaces.Set{}

// Ifaces returns the run's delegation interface set, tolerating a nil context
// or unset field (as in fragment-level unit tests) by returning a zero Set,
// which resolves no delegation targets.
func (c *GenContext) Ifaces() *interfaces.Set {
	if c == nil || c.Interfaces == nil {
		return zeroInterfaceSet
	}
	return c.Interfaces
}

// QualifiedTypeName renders v's Go type name, package-qualified (and the import
// registered with the namer) when v lives outside the target package.
func (c *GenContext) QualifiedTypeName(v gentypes.ValRep) string {
	tn := v.TypeName()
	if c.TargetPackage == v.PackagePath() || v.PackagePath() == "" {
		return tn
	}
	pkg := c.Namer.NameString(v.PackagePath())
	if tn[0:1] == "*" {
		tn = tn[1:]
		pkg = "*" + pkg
	}
	return pkg + "." + tn
}
