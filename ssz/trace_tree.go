package ssz

// Reconstruct the binary merkle tree from the ordered collapse events recorded
// in trace.go, assign generalized indices, and elide zero subtrees — producing a
// node tree byte-identical (modulo the semantic overlay) to the
// Python/remerkleable reference tracer.
//
// The reconstruction is a stack-based pass exploiting buffer containment: a
// child's collapse lands its root chunk at a known offset before the enclosing
// collapse runs, so events arrive in post-order and pending results can be
// matched to parents by buffer position.
//
// The per-op tree shapes and zero-elision exactly mirror remerkleable's
// tree.subtree_fill_to_contents / progressive.subtree_fill_progressive and its
// zero_node virtualization (see remerkleable/tree.py, progressive.py):
//   - balanced merkleize  -> subtree_fill_to_contents(slots, depth)
//   - progressive         -> subtree_fill_progressive(slots, 0) with windows of
//                            capacity 1,4,16,... at depths 0,2,4,...
//   - zero_node(d): d>=1 -> an elided `zero: d` subtree (root == zeroHashes[d]);
//                   d==0 -> a plain all-zero leaf chunk (not elided), matching the
//                           reference tracer's `if d:` guard.

import (
	"encoding/binary"
	"fmt"

	"github.com/minio/sha256-simd"
)

// TraceNode is one node of the reconstructed merkle tree. Exactly one shape is
// set: leaf (a 32-byte chunk), zero (an elided all-zero subtree of ZeroDepth),
// or branch (Left+Right). Root is always the node's merkle root; G is its
// global generalized index (root = 1, children 2g / 2g+1).
type TraceNode struct {
	G    uint64
	Root [32]byte

	// exactly one shape:
	Leaf      bool // 32-byte chunk; Root == the bytes
	Zero      bool // elided zero subtree; Root == zeroHashes[ZeroDepth], ZeroDepth >= 1
	ZeroDepth int
	Left      *TraceNode
	Right     *TraceNode

	// resultPos is the buffer position of this subtree's collapsed root chunk;
	// used only during reconstruction to match children to parents.
	resultPos int
}

func hashPair(l, r [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], l[:])
	copy(buf[32:], r[:])
	return sha256.Sum256(buf[:])
}

// leafNode builds a leaf from up to 32 bytes (zero-right-padded to a full chunk).
func leafNode(b []byte) *TraceNode {
	n := &TraceNode{Leaf: true}
	copy(n.Root[:], b)
	return n
}

// zeroSubtree mirrors remerkleable's zero_node(depth): depth 0 is a plain
// all-zero leaf chunk (the reference tracer renders it as a `leaf`, not `zero`),
// while depth >= 1 is an elided zero subtree rendered as `zero: depth`.
func zeroSubtree(depth int) *TraceNode {
	if depth == 0 {
		return &TraceNode{Leaf: true} // Root is all-zero
	}
	return &TraceNode{Zero: true, ZeroDepth: depth, Root: zeroHashes[depth]}
}

func branchNode(l, r *TraceNode) *TraceNode {
	return &TraceNode{Left: l, Right: r, Root: hashPair(l.Root, r.Root)}
}

// fillToContents mirrors remerkleable tree.subtree_fill_to_contents: a balanced
// binary tree of the given depth whose leftmost leaves are `nodes` and whose
// empty regions collapse to zero_node subtrees.
func fillToContents(nodes []*TraceNode, depth int) *TraceNode {
	if len(nodes) == 0 {
		return zeroSubtree(depth)
	}
	if depth == 0 {
		// len(nodes) must be 1 here (guaranteed by the callers / merkleize limits).
		return nodes[0]
	}
	pivot := 1 << (depth - 1)
	if len(nodes) <= pivot {
		return branchNode(fillToContents(nodes, depth-1), zeroSubtree(depth-1))
	}
	return branchNode(fillToContents(nodes[:pivot], depth-1), fillToContents(nodes[pivot:], depth-1))
}

// fillProgressive mirrors remerkleable progressive.subtree_fill_progressive: a
// right-leaning spine of balanced windows of capacity 1, 4, 16, ... (depths
// 0, 2, 4, ...). An exhausted spine terminates in a single depth-0 zero chunk.
func fillProgressive(nodes []*TraceNode, depth int) *TraceNode {
	if len(nodes) == 0 {
		return zeroSubtree(0) // terminal tail: one zero chunk, depth 0
	}
	base := 1 << depth
	window, rest := nodes, []*TraceNode(nil)
	if len(nodes) > base {
		window, rest = nodes[:base], nodes[base:]
	}
	left := fillToContents(window, depth)
	right := fillProgressive(rest, depth+2)
	return branchNode(left, right)
}

// mixinLeaf builds a length/selector mix-in leaf: num little-endian in a 32-byte
// chunk (uint256_le for values < 2^64), matching MerkleizeWithMixin.
func mixinLeaf(num uint64) *TraceNode {
	var b [32]byte
	binary.LittleEndian.PutUint64(b[:8], num)
	return &TraceNode{Leaf: true, Root: b}
}

// activeFieldsLeaf builds a progressive-container active-fields mix-in leaf: the
// packed bitvector copied into a 32-byte chunk, matching
// MerkleizeProgressiveWithActiveFields.
func activeFieldsLeaf(af []byte) *TraceNode {
	var b [32]byte
	copy(b[:], af)
	return &TraceNode{Leaf: true, Root: b}
}

// buildEventNode constructs the subtree an event collapses to, given its
// resolved child slots (each slot is either a reconstructed child subtree or a
// leaf chunk from the event's input copy).
func buildEventNode(e traceEvent, slots []*TraceNode) (*TraceNode, error) {
	switch e.op {
	case opMerkleize:
		// limit 0 => balanced tree of depth(chunkCount).
		return fillToContents(slots, int(depth(uint64(len(slots))))), nil
	case opMerkleizeMixin:
		content := fillToContents(slots, int(depth(e.limit)))
		return branchNode(content, mixinLeaf(e.num)), nil
	case opProgressive:
		return fillProgressive(slots, 0), nil
	case opProgressiveMixin:
		return branchNode(fillProgressive(slots, 0), mixinLeaf(e.num)), nil
	case opProgressiveActiveFields:
		return branchNode(fillProgressive(slots, 0), activeFieldsLeaf(e.activeFields)), nil
	default:
		return nil, fmt.Errorf("trace reconstruct: unknown op %d", e.op)
	}
}

// reconstruct rebuilds the merkle tree from the ordered collapse events. Events
// are in post-order (children before parents); a stack of completed subtrees
// keyed by result buffer position resolves parent/child nesting.
func reconstruct(events []traceEvent) (*TraceNode, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("trace reconstruct: no events (tracing not enabled?)")
	}

	type pending struct {
		pos int
		n   *TraceNode
	}
	var stack []pending

	for _, e := range events {
		chunkCount := (len(e.input) + 31) / 32
		end := e.bufStart + len(e.input)

		// Pop every pending subtree whose result lands within this event's input
		// range; by strict LIFO nesting these are exactly the direct children and
		// sit contiguously at the top of the stack.
		childByPos := make(map[int]*TraceNode)
		for len(stack) > 0 && stack[len(stack)-1].pos >= e.bufStart {
			p := stack[len(stack)-1]
			if p.pos >= end {
				return nil, fmt.Errorf("trace reconstruct: pending result at %d beyond event input [%d,%d)", p.pos, e.bufStart, end)
			}
			stack = stack[:len(stack)-1]
			childByPos[p.pos] = p.n
		}

		slots := make([]*TraceNode, chunkCount)
		for i := 0; i < chunkCount; i++ {
			pos := e.bufStart + 32*i
			if c, ok := childByPos[pos]; ok {
				slots[i] = c
				continue
			}
			lo, hi := 32*i, 32*i+32
			if hi > len(e.input) {
				hi = len(e.input)
			}
			slots[i] = leafNode(e.input[lo:hi])
		}

		n, err := buildEventNode(e, slots)
		if err != nil {
			return nil, err
		}
		n.resultPos = e.bufStart
		stack = append(stack, pending{pos: e.bufStart, n: n})
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("trace reconstruct: expected a single root, got %d pending subtrees", len(stack))
	}
	root := stack[0].n
	assignGindex(root, 1)
	return root, nil
}

// assignGindex walks the tree top-down assigning global generalized indices:
// root = 1, left child 2g, right child 2g+1. Uniform for balanced and
// progressive subtrees alike (the progressive spine is an ordinary right-leaning
// branch chain).
func assignGindex(n *TraceNode, g uint64) {
	n.G = g
	if n.Left != nil {
		assignGindex(n.Left, 2*g)
		assignGindex(n.Right, 2*g+1)
	}
}

// verify recomputes the node's merkle root purely from the emitted tree shape,
// independent of the roots cached during construction. Used to self-check the
// reconstruction against the value's HashTreeRoot.
func (n *TraceNode) verify() [32]byte {
	if n.Zero {
		return zeroHashes[n.ZeroDepth]
	}
	if n.Leaf {
		return n.Root
	}
	return hashPair(n.Left.verify(), n.Right.verify())
}
