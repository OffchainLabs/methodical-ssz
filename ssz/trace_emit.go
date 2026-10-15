package ssz

// Drive a value's HashTreeRootWith with tracing enabled, reconstruct the tree
// (trace_tree.go), self-verify, and emit the trace as YAML — plus a flat
// g -> root map that is the primary mechanical-bisection view (diff two flat
// maps, take the highest g present in both with differing root).
//
// Determinism: lowercase 0x hex, exactly 64 hex chars per root, fixed key order,
// zero subtrees elided. The structural tree + generalized indices are fully
// determined by the SSZ type + value, so two correct libraries emit identical
// flat maps.

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
)

// TraceResult bundles a reconstructed trace with its self-verified root.
type TraceResult struct {
	Type string     // type_repr of the traced value (caller-supplied; may be empty)
	Root [32]byte   // the value's hash_tree_root
	Tree *TraceNode // root node of the reconstructed merkle tree (G == 1)
}

// TraceValue enables tracing on a fresh Hasher, drives v.HashTreeRootWith,
// reconstructs the merkle tree from the recorded collapse events, and
// self-verifies the reconstruction against the value's root. typeName is an
// optional label for the envelope (e.g. "BeaconBlockBody").
func TraceValue(v HashRoot, typeName string) (*TraceResult, error) {
	h := NewHasher()
	h.EnableTrace()
	if err := v.HashTreeRootWith(h); err != nil {
		return nil, fmt.Errorf("trace: HashTreeRootWith: %w", err)
	}
	root, err := h.HashRoot()
	if err != nil {
		return nil, fmt.Errorf("trace: HashRoot: %w", err)
	}

	tree, err := reconstruct(h.TraceEvents())
	if err != nil {
		return nil, err
	}
	if got := tree.verify(); got != root {
		return nil, fmt.Errorf("trace self-verify mismatch: reconstructed %x != hash_tree_root %x", got, root)
	}
	if tree.Root != root {
		return nil, fmt.Errorf("trace root mismatch: tree root %x != hash_tree_root %x", tree.Root, root)
	}
	return &TraceResult{Type: typeName, Root: root, Tree: tree}, nil
}

func hexRoot(b [32]byte) string {
	return "0x" + hex.EncodeToString(b[:])
}

// WriteYAML emits the nested trace document (envelope + recursive node tree).
func (r *TraceResult) WriteYAML(w io.Writer) error {
	var sb strings.Builder
	sb.WriteString("trace_version: 1\n")
	fmt.Fprintf(&sb, "type: %q\n", r.Type)
	fmt.Fprintf(&sb, "root: %q\n", hexRoot(r.Root))
	sb.WriteString("hash: sha256\n")
	sb.WriteString("endianness: little\n")
	sb.WriteString("elide_zero_subtrees: true\n")
	sb.WriteString("tree:\n")
	emitNode(&sb, r.Tree, 1)
	_, err := io.WriteString(w, sb.String())
	return err
}

// emitNode writes one node at the given indent (in 2-space units). Key order and
// shape: leaf carries only `leaf` (leaf == root); zero and branch carry `root`;
// branch nests `left`/`right`.
func emitNode(sb *strings.Builder, n *TraceNode, indent int) {
	pad := strings.Repeat("  ", indent)
	fmt.Fprintf(sb, "%sg: %d\n", pad, n.G)
	switch {
	case n.Leaf:
		fmt.Fprintf(sb, "%sleaf: %q\n", pad, hexRoot(n.Root))
	case n.Zero:
		fmt.Fprintf(sb, "%sroot: %q\n", pad, hexRoot(n.Root))
		fmt.Fprintf(sb, "%szero: %d\n", pad, n.ZeroDepth)
	default:
		fmt.Fprintf(sb, "%sroot: %q\n", pad, hexRoot(n.Root))
		fmt.Fprintf(sb, "%sleft:\n", pad)
		emitNode(sb, n.Left, indent+1)
		fmt.Fprintf(sb, "%sright:\n", pad)
		emitNode(sb, n.Right, indent+1)
	}
}

// WriteFlat emits the flat g -> root map of every (non-expanded) node, sorted by
// generalized index. This is the mechanical bisection view: elided zero subtrees
// contribute a single line (their descendants are not present), and both correct
// libraries produce the identical g set, so the highest g whose root differs is
// the divergence frontier.
func (r *TraceResult) WriteFlat(w io.Writer) error {
	flat := make(map[uint64][32]byte)
	collectFlat(r.Tree, flat)

	gs := make([]uint64, 0, len(flat))
	for g := range flat {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i] < gs[j] })

	var sb strings.Builder
	sb.WriteString("trace_version: 1\n")
	fmt.Fprintf(&sb, "type: %q\n", r.Type)
	fmt.Fprintf(&sb, "root: %q\n", hexRoot(r.Root))
	sb.WriteString("nodes:\n")
	for _, g := range gs {
		root := flat[g]
		fmt.Fprintf(&sb, "  %d: %q\n", g, hexRoot(root))
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func collectFlat(n *TraceNode, out map[uint64][32]byte) {
	out[n.G] = n.Root
	if n.Left != nil {
		collectFlat(n.Left, out)
		collectFlat(n.Right, out)
	}
}

// FlatMap returns the g -> root map directly (for programmatic bisection).
func (r *TraceResult) FlatMap() map[uint64][32]byte {
	flat := make(map[uint64][32]byte)
	collectFlat(r.Tree, flat)
	return flat
}
