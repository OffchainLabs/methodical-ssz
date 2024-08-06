package interfaces

import (
	"fmt"
	"go/types"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// Set holds the delegation-target interfaces for one generation run. The
// pointer values are identity keys, not just handles: ValRep.Interfaces maps
// are built with them as keys (see SupportMap), and the ops look types up via
// ValRep.SatisfiesInterface using the same pointers. All keys for a run must
// therefore come from a single Set instance — the frontend (which builds the
// support maps) and the render GenContext (which supplies the lookup keys)
// must share one.
//
// A Set is constructed either from methodical's own ssz package (DefaultSet,
// shared singleton) or from any other load universe / interface source
// (NewSet) — which is what makes delegation pluggable: a codegen plugin can
// target its own interfaces by supplying its own Set. The zero Set disables
// delegation (every field is nil, so no op resolves a delegation target).
type Set struct {
	Marshaler   *types.Interface
	Unmarshaler *types.Interface
	FullHasher  *types.Interface
	LightHasher *types.Interface
	Sizer       *types.Interface
}

var (
	defaultSet     *Set
	defaultSetErr  error
	defaultSetOnce sync.Once
)

// DefaultSet returns the process-wide Set built from methodical's own ssz
// package, loading it once. Callers that need identity-consistent keys across
// the frontend and render phases get them by both using this singleton.
func DefaultSet() (*Set, error) {
	defaultSetOnce.Do(func() {
		pkg, err := loadSsz()
		if err != nil {
			defaultSetErr = err
			return
		}
		defaultSet, defaultSetErr = NewSet(pkg.Types)
	})
	return defaultSet, defaultSetErr
}

// NewSet builds a Set from an instance of the ssz runtime package — typically
// the one in a generation target's own load universe, or methodical's own via
// DefaultSet. Each delegation interface is declared in the runtime package and
// looked up independently (Sizer and LightHasher are the declared single-method
// subsets of Marshaler and HashRoot).
func NewSet(tp *types.Package) (*Set, error) {
	s := &Set{}
	for name, dst := range map[string]**types.Interface{
		"Marshaler":   &s.Marshaler,
		"Unmarshaler": &s.Unmarshaler,
		"HashRoot":    &s.FullHasher,
		"LightHasher": &s.LightHasher,
		"Sizer":       &s.Sizer,
	} {
		iface, err := lookupInterface(tp, name)
		if err != nil {
			return nil, err
		}
		*dst = iface
	}
	return s, nil
}

// lookupInterface resolves a declared interface type by name from the package
// scope.
func lookupInterface(tp *types.Package, name string) (*types.Interface, error) {
	obj := tp.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("interface %s not found in package %s", name, tp.Path())
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%s in package %s is not an interface", name, tp.Path())
	}
	return iface, nil
}

// SupportMap reports which of the Set's interfaces t satisfies (with a value
// or pointer receiver), keyed by the Set's identity pointers.
func (s *Set) SupportMap(t types.Type) map[*types.Interface]bool {
	return map[*types.Interface]bool{
		s.Marshaler:   types.Implements(t, s.Marshaler) || types.Implements(types.NewPointer(t), s.Marshaler),
		s.Unmarshaler: types.Implements(t, s.Unmarshaler) || types.Implements(types.NewPointer(t), s.Unmarshaler),
		s.LightHasher: types.Implements(t, s.LightHasher) || types.Implements(types.NewPointer(t), s.LightHasher),
		s.FullHasher:  s.implementsFullHasher(t),
		s.Sizer:       types.Implements(t, s.Sizer) || types.Implements(types.NewPointer(t), s.Sizer),
	}
}

// implementsFullHasher reports whether t (or *t) provides the full hasher
// interface (ssz.HashRoot: HashTreeRoot plus HashTreeRootWith(*ssz.Hasher)).
//
// Because HashRoot mentions the named type *ssz.Hasher, and go/types compares
// named types by object identity (which never holds across a different
// packages.Load), the Set's FullHasher cannot be compared against a candidate
// from another load universe. Instead, resolve the HashRoot interface from t's
// own load universe and run types.Implements against that. The Set's FullHasher
// remains the identity key in the support map; only the comparison interface is
// universe-local.
func (s *Set) implementsFullHasher(t types.Type) bool {
	iface := hashRootFor(t)
	if iface == nil {
		// t's universe has no ssz package (so no signature in it can mention
		// *ssz.Hasher), or t is not a named type: fall back to the Set's own,
		// which is correct for same-universe types.
		iface = s.FullHasher
	}
	return types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface)
}

const sszPkgPath = "github.com/OffchainLabs/methodical-ssz/ssz"

// hashRootFor finds the ssz.HashRoot interface in the load universe of the
// package that declares t, or nil if t is unnamed or its universe does not
// include the ssz package.
func hashRootFor(t types.Type) *types.Interface {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return nil
	}
	sszPkg := findImport(named.Obj().Pkg(), sszPkgPath, make(map[*types.Package]bool))
	if sszPkg == nil {
		return nil
	}
	obj := sszPkg.Scope().Lookup("HashRoot")
	if obj == nil {
		return nil
	}
	iface, _ := obj.Type().Underlying().(*types.Interface)
	return iface
}

func findImport(p *types.Package, path string, seen map[*types.Package]bool) *types.Package {
	if p == nil || seen[p] {
		return nil
	}
	seen[p] = true
	if p.Path() == path {
		return p
	}
	for _, imp := range p.Imports() {
		if found := findImport(imp, path, seen); found != nil {
			return found
		}
	}
	return nil
}

func loadSsz() (*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedTypes}, sszPkgPath)
	if err != nil {
		return nil, errors.Wrap(err, "error from packages.Load for methodical-ssz")
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("missing package, add github.com/OffchainLabs/methodical-ssz to your go.mod")
	}
	for _, p := range pkgs {
		if p.ID == sszPkgPath {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%s not found in go package index - add to your go.mod", sszPkgPath)
}
