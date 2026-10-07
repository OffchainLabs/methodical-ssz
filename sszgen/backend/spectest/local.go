package spectest

import (
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/OffchainLabs/methodical-ssz/specs"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// DefaultLocalPackageName is the package name written into the generated test
// file when the gen-spectest subcommand is not given an override.
const DefaultLocalPackageName = "spectest"

// WriteLocalSpecTestFiles writes a standalone, test-only package meant to live
// inside the module that owns the types. A per-type `package` key lets one
// generated package exercise types spread across several Go packages. Nothing is
// cloned or generated from the types, so the tool needs no access to their source.
func WriteLocalSpecTestFiles(cases map[specs.TestIdent]specs.Fixture, rels *SpecRelationships, fs afero.Fs, pkgName string) error {
	if pkgName == "" {
		pkgName = DefaultLocalPackageName
	}
	if rels.Package == "" {
		return errors.New("spec test config is missing the package key")
	}

	// only packages whose types matched a fixture get an import — an unused
	// import would fail compilation of the generated file
	used, err := usedPackages(cases, rels)
	if err != nil {
		return err
	}
	aliases := assignAliases(used)

	caseFuncs, err := renderCaseFuncs(cases, rels, fs, aliases)
	if err != nil {
		return err
	}

	packageDecl := "package " + pkgName + "\n\n"
	contents := packageDecl + localImportsBlock(aliases) + "\n\n" + strings.Join(caseFuncs, "\n\n")

	testBytes, err := format.Source([]byte(contents))
	if err != nil {
		return err
	}

	fname := "methodical_test.go"
	if err := afero.WriteFile(fs, fname, testBytes, 0666); err != nil {
		return errors.Wrapf(err, "error writing spectest functions to %s", fname)
	}
	return nil
}

// usedPackages performs the same case-to-type matching as renderCaseFuncs and
// returns the set of packages declaring at least one matched type.
func usedPackages(cases map[specs.TestIdent]specs.Fixture, rels *SpecRelationships) (map[string]bool, error) {
	used := make(map[string]bool)
	fg := specs.GroupByFork(cases)
	for _, fork := range specs.ForkOrder {
		raf, err := rels.RelationsAtFork(fork)
		if err != nil {
			return nil, err
		}
		for _, id := range fg[fork] {
			if rt, ok := raf[id.Name]; ok {
				used[rt.Package] = true
			}
		}
	}
	return used, nil
}

// assignAliases gives each used package a deterministic, collision-free import
// alias: the candidate is the sanitized last path element, falling back to the
// previous element prepended, then to a numeric suffix. Sorted processing keeps
// the output stable across runs.
func assignAliases(used map[string]bool) map[string]string {
	paths := make([]string, 0, len(used))
	for p := range used {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	taken := make(map[string]bool)
	aliases := make(map[string]string, len(paths))
	for _, p := range paths {
		parts := strings.Split(p, "/")
		cands := []string{sanitizeAlias(parts[len(parts)-1])}
		if len(parts) > 1 {
			cands = append(cands, sanitizeAlias(parts[len(parts)-2]+parts[len(parts)-1]))
		}
		alias := ""
		for _, c := range cands {
			if c != "" && !taken[c] {
				alias = c
				break
			}
		}
		for n := 2; alias == ""; n++ {
			c := fmt.Sprintf("%s%d", cands[0], n)
			if !taken[c] {
				alias = c
			}
		}
		taken[alias] = true
		aliases[p] = alias
	}
	return aliases
}

// sanitizeAlias makes a path element usable as an import alias identifier.
func sanitizeAlias(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
	if s != "" && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	return s
}

// localImportsBlock renders the import block: the fixed test imports plus one
// aliased import per used types package, in sorted import-path order.
func localImportsBlock(aliases map[string]string) string {
	paths := make([]string, 0, len(aliases))
	for p := range aliases {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	b := &strings.Builder{}
	b.WriteString("import (\n\t\"bytes\"\n\t\"testing\"\n\n\t\"github.com/OffchainLabs/methodical-ssz/specs\"\n")
	if len(paths) > 0 {
		b.WriteString("\n")
		for _, p := range paths {
			fmt.Fprintf(b, "\t%s %q\n", aliases[p], p)
		}
	}
	b.WriteString(")")
	return b.String()
}
