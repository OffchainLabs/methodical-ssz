package core

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// ImportNamer assigns the identifiers generated code uses to qualify
// cross-package references, and renders the matching import block.
//
// Imports are stored in a tree keyed by path segment: the fully qualified
// import path is split on "/" and a node is inserted at the end of that walk,
// so every package is identified by its full path — two packages can never
// clobber each other no matter how their aliases or base names relate.
// Stdlib packages simply sit at (or near) the root. Assigned names map
// back to the node that owns them, and name collisions are resolved by
// walking up the tree, extending the candidate with ancestor segments.
type ImportNamer struct {
	source string
	root   *importNode
	// names maps every claimed identifier to the node that owns it — the
	// alias map pointing back into the tree.
	names map[string]*importNode
}

// importNode is one path segment in the import tree. A node with imported set
// is an actual imported package (the end of a full path walk); other nodes
// are intermediate segments.
type importNode struct {
	segment  string
	parent   *importNode
	children map[string]*importNode
	depth    int
	imported bool
	// used marks packages whose name has been handed to code generation
	// (NameString) — only those are emitted by ImportPairs, so reserving more
	// packages than the generated code references is harmless. A used node's
	// name is frozen: it may already appear in generated text, so the
	// shortest-path-wins reorganization never steals from it.
	used bool
	// name is the identifier generated code uses for this package. bare means
	// the import is emitted without an explicit alias (the identifier is the
	// package's own name, as with unaliased stdlib defaults like "fmt").
	name string
	bare bool
}

// path reconstructs the full import path by walking back to the root.
func (n *importNode) path() string {
	segments := make([]string, 0, 4)
	for cur := n; cur.parent != nil; cur = cur.parent {
		segments = append(segments, cur.segment)
	}
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}
	return strings.Join(segments, "/")
}

// insert walks p's segments from the root, creating nodes as needed, and
// returns the terminal node.
func (n *ImportNamer) insert(p string) *importNode {
	cur := n.root
	for _, seg := range strings.Split(p, "/") {
		if cur.children == nil {
			cur.children = make(map[string]*importNode)
		}
		child, ok := cur.children[seg]
		if !ok {
			child = &importNode{segment: seg, parent: cur, depth: cur.depth + 1}
			cur.children[seg] = child
		}
		cur = child
	}
	return cur
}

var pbRuntime = "google.golang.org/protobuf/runtime/protoimpl"
var protoAliases = map[string]string{
	pbRuntime: "protoimpl",
	"google.golang.org/protobuf/internal/impl": "protoimpl",
}

// sanitizeSegment makes a path segment usable inside a go identifier: domain
// dots (example.com) and dashes (go-bitfield) become underscores.
func sanitizeSegment(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ReplaceAll(s, "-", "_")
}

// NameString returns the identifier generated code should use to qualify
// references to package path p, registering the import on first use. The
// candidate name is the sanitized last path segment; on collision with a name
// owned by another package, the candidate is extended with ancestor segments
// (engine/v1 vs widget/v1 → v1, widget_v1).
func (n *ImportNamer) NameString(p string) string {
	if alias, isProto := protoAliases[p]; isProto {
		nd := n.insert(pbRuntime)
		nd.imported = true
		nd.used = true
		nd.name = alias
		n.names[alias] = nd
		return alias
	}
	// no package name for self
	if p == n.source {
		return ""
	}
	nd := n.insert(p)
	if !nd.imported {
		nd.imported = true
		n.assignName(nd)
	}
	nd.used = true
	return nd.name
}

// Reserve registers p ahead of code generation without marking it used.
// Reserved packages participate in naming — with the shortest-path-wins
// reorganization, so e.g. stdlib "fmt" takes the bare name from a reserved
// "github.com/example/fmt" regardless of arrival order — but are only emitted
// in the import block if generation later requests their name (NameString).
func (n *ImportNamer) Reserve(p string) {
	if p == "" || p == n.source {
		return
	}
	if _, isProto := protoAliases[p]; isProto {
		// route through NameString's normalization lazily; the fixed alias
		// cannot collide with path-derived names in practice
		return
	}
	nd := n.insert(p)
	if nd.imported {
		return
	}
	nd.imported = true
	n.assignName(nd)
}

// assignName picks nd's identifier: the sanitized last path segment, extended
// with ancestor segments on collision. Shortest path wins: if the candidate is
// owned by a deeper node whose name has not been used in generated text yet,
// the owner is renamed (recursively) and nd takes the shorter name. Steal
// chains terminate because each victim is strictly deeper than the thief.
func (n *ImportNamer) assignName(nd *importNode) {
	candidate := sanitizeSegment(nd.segment)
	for cur := nd.parent; ; cur = cur.parent {
		owner, taken := n.names[candidate]
		if !taken || owner == nd {
			nd.name = candidate
			n.names[candidate] = nd
			return
		}
		if !owner.used && owner.depth > nd.depth {
			nd.name = candidate
			n.names[candidate] = nd
			n.assignName(owner)
			return
		}
		if cur == nil || cur.parent == nil {
			panic(fmt.Sprintf("unable to find unique name for package %s", nd.path()))
		}
		candidate = sanitizeSegment(cur.segment) + "_" + candidate
	}
}

func (n *ImportNamer) Name(p *types.Package) string {
	return n.NameString(p.Path())
}

// NumAliases reports how many packages have been registered (defaults plus any
// accumulated during generation). Primarily useful for tests asserting which
// cross-package imports a method set pulled in.
func (n *ImportNamer) NumAliases() int {
	count := 0
	n.walk(func(nd *importNode) { count++ })
	return count
}

// walk visits every used node in deterministic (sorted-segment) order.
func (n *ImportNamer) walk(visit func(*importNode)) {
	var rec func(*importNode)
	rec = func(nd *importNode) {
		if nd.used {
			visit(nd)
		}
		segs := make([]string, 0, len(nd.children))
		for s := range nd.children {
			segs = append(segs, s)
		}
		sort.Strings(segs)
		for _, s := range segs {
			rec(nd.children[s])
		}
	}
	rec(n.root)
}

// ImportPairs renders one import line per registered package: `name "path"`,
// or `"path"` for bare imports. Lines are emitted in deterministic tree order;
// the final ordering of the generated file's import block is gofmt's
// (format.Source sorts import specs by path).
func (n *ImportNamer) ImportPairs() string {
	lines := make([]string, 0)
	n.walk(func(nd *importNode) {
		if nd.bare {
			lines = append(lines, fmt.Sprintf("%q", nd.path()))
			return
		}
		lines = append(lines, fmt.Sprintf("%s %q", nd.name, nd.path()))
	})
	return strings.Join(lines, "\n")
}

// ImportSource renders a complete import declaration block (see ImportPairs).
func (n *ImportNamer) ImportSource() string {
	return fmt.Sprintf("import (\n%s\n)\n", n.ImportPairs())
}

// NewImportNamer builds a namer for generating code into the source package.
// defaults seeds always-emitted imports, keyed by package path; an empty alias
// means the import is bare (no explicit alias) and the package's base name is
// claimed as its identifier.
func NewImportNamer(source string, defaults map[string]string) *ImportNamer {
	n := &ImportNamer{
		source: source,
		root:   &importNode{},
		names:  make(map[string]*importNode),
	}
	// seed in sorted order so name claims are deterministic
	paths := make([]string, 0, len(defaults))
	for p := range defaults {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		alias := defaults[p]
		nd := n.insert(p)
		nd.imported = true
		// defaults are referenced unconditionally by the body templates, so
		// they are always emitted
		nd.used = true
		if alias == "" {
			nd.bare = true
			nd.name = sanitizeSegment(nd.segment)
		} else {
			nd.name = alias
		}
		n.names[nd.name] = nd
	}
	return n
}
