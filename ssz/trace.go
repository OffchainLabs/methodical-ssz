package ssz

// This file adds optional merkleization tracing to the flat-buffer Hasher.
//
// When tracing is enabled (Hasher.EnableTrace), every collapse operation records
// a traceEvent capturing the input chunks (a copy of h.buf[indx:] taken before
// the collapse) plus the op-specific parameters. The ordered event list fully
// encodes the merkle tree; reconstruction lives in trace_tree.go.
//
// Tracing is nil by default and adds zero overhead to the untraced path.

// traceOp identifies which collapse produced a subtree root.
type traceOp uint8

const (
	opMerkleize               traceOp = iota // balanced merkleize, buf[indx:] with `limit` (0 => next_pow2(chunkCount))
	opMerkleizeMixin                         // balanced + mix_in_length(num), padded to next_pow2(limit)
	opProgressive                            // merkleize_progressive
	opProgressiveMixin                       // progressive + mix_in_length(num)
	opProgressiveActiveFields                // progressive + mix_in_active_fields(activeFields)
)

// traceEvent records one collapse of h.buf[bufStart:] into a single root chunk.
type traceEvent struct {
	op       traceOp
	bufStart int    // h.buf index where the input began; also where the result lands
	input    []byte // copy of h.buf[bufStart:] at collapse time (always chunk-aligned)

	// op-specific parameters:
	num          uint64 // length mix-in (opMerkleizeMixin, opProgressiveMixin)
	limit        uint64 // balanced chunk-count limit (opMerkleize: 0 => next_pow2; opMerkleizeMixin: explicit)
	activeFields []byte // packed active-fields bitvector (opProgressiveActiveFields)
}

// traceState accumulates the ordered events of a single HashTreeRoot pass.
type traceState struct {
	events []traceEvent
}

// EnableTrace turns on merkleization tracing for subsequent hashing on h. Call
// before driving a value's HashTreeRootWith; retrieve the events with
// TraceEvents after HashRoot. Reset() clears the trace.
func (h *Hasher) EnableTrace() { h.trace = &traceState{} }

// TracingEnabled reports whether tracing is active.
func (h *Hasher) TracingEnabled() bool { return h.trace != nil }

// TraceEvents returns the recorded collapse events in order (nil if tracing off).
func (h *Hasher) TraceEvents() []traceEvent {
	if h.trace == nil {
		return nil
	}
	return h.trace.events
}

// record appends an event, snapshotting the current input chunks. Callers pass
// the already-copied input so the snapshot is taken before the collapse mutates
// h.buf.
func (h *Hasher) record(op traceOp, bufStart int, num, limit uint64, activeFields []byte) {
	if h.trace == nil {
		return
	}
	input := make([]byte, len(h.buf)-bufStart)
	copy(input, h.buf[bufStart:])
	var af []byte
	if activeFields != nil {
		af = make([]byte, len(activeFields))
		copy(af, activeFields)
	}
	h.trace.events = append(h.trace.events, traceEvent{
		op:           op,
		bufStart:     bufStart,
		input:        input,
		num:          num,
		limit:        limit,
		activeFields: af,
	})
}
