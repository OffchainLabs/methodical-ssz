package ssz

import (
	"encoding/binary"
	"fmt"
	"hash"
	"math/bits"
	"sync"

	"github.com/minio/sha256-simd"
	"github.com/prysmaticlabs/gohashtree"
)

// LightHasher is the interface implemented by types that can compute their own
// hash tree root. It is the subset of HashRoot used by hashing delegation when
// a type does not provide the Hasher-composing HashTreeRootWith form.
type LightHasher interface {
	HashTreeRoot() ([32]byte, error)
}

type HashRoot interface {
	LightHasher
	HashTreeRootWith(hh *Hasher) error
}

var zeroHashes [65][32]byte
var zeroHashLevels map[string]int
var trueBytes, falseBytes []byte

const (
	mask0 = ^uint64((1 << (1 << iota)) - 1)
	mask1
	mask2
	mask3
	mask4
	mask5
)

const (
	bit0 = uint8(1 << iota)
	bit1
	bit2
	bit3
	bit4
	bit5
)

func init() {
	falseBytes = make([]byte, 32)
	trueBytes = make([]byte, 32)
	trueBytes[0] = 1
	zeroHashLevels = make(map[string]int)
	zeroHashLevels[string(falseBytes)] = 0

	tmp := [64]byte{}
	for i := 0; i < 64; i++ {
		copy(tmp[:32], zeroHashes[i][:])
		copy(tmp[32:], zeroHashes[i][:])
		zeroHashes[i+1] = sha256.Sum256(tmp[:])
		zeroHashLevels[string(zeroHashes[i+1][:])] = i + 1
	}
}

// HashWithDefaultHasher hashes a HashRoot object with a Hasher from
// the default HasherPool
func HashWithDefaultHasher(v HashRoot) ([32]byte, error) {
	hh := DefaultHasherPool.Get()
	if err := v.HashTreeRootWith(hh); err != nil {
		DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}
	root, err := hh.HashRoot()
	DefaultHasherPool.Put(hh)
	return root, err
}

var zeroBytes = make([]byte, 32)

// DefaultHasherPool is a default hasher pool
var DefaultHasherPool HasherPool

// Hasher is a utility tool to hash SSZ structs
type Hasher struct {
	buf []byte

	// tmp array used for uint64 and bitlist processing
	tmp []byte

	// tmp array used during the merkleize process
	merkleizeTmp []byte

	hash hash.Hash

	// trace, when non-nil, records each collapse for merkleization tracing
	// (see trace.go). nil on the hot path => zero overhead.
	trace *traceState
}

// NewHasher creates a new Hasher object
func NewHasher() *Hasher {
	return &Hasher{
		hash: sha256.New(),
		tmp:  make([]byte, 32),
	}
}

// NewHasherWithHash creates a new Hasher object with a custom hash function
func NewHasherWithHash(hh hash.Hash) *Hasher {
	return &Hasher{
		hash: hh,
		tmp:  make([]byte, 32),
	}
}

// HashRoot creates the hash final hash root
func (h *Hasher) HashRoot() (res [32]byte, err error) {
	if len(h.buf) != 32 {
		err = fmt.Errorf("expected 32 byte size")
		return
	}
	copy(res[:], h.buf)
	return
}

// Reset resets the Hasher obj
func (h *Hasher) Reset() {
	h.buf = h.buf[:0]
	h.hash.Reset()
	h.trace = nil
}

func (h *Hasher) Index() int {
	return len(h.buf)
}

func (h *Hasher) Append(i []byte) {
	h.buf = append(h.buf, i...)
}

func (h *Hasher) AppendUint8(i uint8) {
	h.buf = append(h.buf, i)
}

// AppendBool appends a boolean as a single byte, for packed bool collections
// (PutBool, by contrast, pads to a full chunk for scalar fields).
func (h *Hasher) AppendBool(b bool) {
	if b {
		h.buf = append(h.buf, 1)
	} else {
		h.buf = append(h.buf, 0)
	}
}

func (h *Hasher) AppendUint16(i uint16) {
	h.buf = binary.LittleEndian.AppendUint16(h.buf, i)
}

func (h *Hasher) AppendUint32(i uint32) {
	h.buf = binary.LittleEndian.AppendUint32(h.buf, i)
}

func (h *Hasher) AppendUint64(i uint64) {
	h.buf = binary.LittleEndian.AppendUint64(h.buf, i)
}

func (h *Hasher) AppendBytes32(b []byte) {
	h.buf = append(h.buf, b...)
	if rest := len(b) % 32; rest != 0 {
		// pad zero bytes to the left
		h.buf = append(h.buf, zeroBytes[:32-rest]...)
	}
}

func (h *Hasher) FillUpTo32() {
	// pad zero bytes to the left
	if rest := len(h.buf) % 32; rest != 0 {
		h.buf = append(h.buf, zeroBytes[:32-rest]...)
	}
}

// PutBool appends a boolean
func (h *Hasher) PutBool(b bool) {
	if b {
		h.buf = append(h.buf, trueBytes...)
	} else {
		h.buf = append(h.buf, falseBytes...)
	}
}

func (h *Hasher) PutBytes(b []byte) {
	if len(b) <= 32 {
		h.AppendBytes32(b)
		return
	}

	// if the bytes are longer than 32 we have to
	// merkleize the content
	indx := h.Index()
	h.AppendBytes32(b)
	h.Merkleize(indx)
}

// PutUint64 appends a uint64 in 32 bytes
func (h *Hasher) PutUint64(i uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	h.AppendBytes32(buf)
}

// PutUint32 appends a uint32 in 32 bytes
func (h *Hasher) PutUint32(i uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, i)
	h.AppendBytes32(buf)
}

// PutUint16 appends a uint16 in 32 bytes
func (h *Hasher) PutUint16(i uint16) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, i)
	h.AppendBytes32(buf)
}

// PutUint8 appends a uint8 in 32 bytes
func (h *Hasher) PutUint8(i uint8) {
	h.AppendBytes32([]byte{byte(i)})
}

// PutUint64Array appends an array of uint64
func (h *Hasher) PutUint64Array(b []uint64, maxCapacity ...uint64) {
	indx := h.Index()
	for _, i := range b {
		h.AppendUint64(i)
	}

	// pad zero bytes to the left
	if rest := len(h.buf) % 32; rest != 0 {
		h.buf = append(h.buf, zeroBytes[:32-rest]...)
	}

	if len(maxCapacity) == 0 {
		// Array with fixed size
		h.Merkleize(indx)
	} else {
		numItems := uint64(len(b))
		limit := CalculateLimit(maxCapacity[0], numItems, 8)

		h.MerkleizeWithMixin(indx, numItems, limit)
	}
}

func CalculateLimit(maxCapacity, numItems, size uint64) uint64 {
	limit := (maxCapacity*size + 31) / 32
	if limit != 0 {
		return limit
	}
	if numItems == 0 {
		return 1
	}
	return numItems
}

// PutBitlist appends a ssz bitlist
func (h *Hasher) PutBitlist(bb []byte, maxSize uint64) {
	var size uint64
	h.tmp, size = parseBitlist(h.tmp[:0], bb)

	// merkleize the content with mix in length
	indx := h.Index()
	h.AppendBytes32(h.tmp)
	h.MerkleizeWithMixin(indx, size, (maxSize+255)/256)
}

func parseBitlist(dst, buf []byte) ([]byte, uint64) {
	msb := uint8(bits.Len8(buf[len(buf)-1])) - 1
	size := uint64(8*(len(buf)-1) + int(msb))

	dst = append(dst, buf...)
	dst[len(dst)-1] &^= uint8(1 << msb)

	newLen := len(dst)
	for i := len(dst) - 1; i >= 0; i-- {
		if dst[i] != 0x00 {
			break
		}
		newLen = i
	}
	res := dst[:newLen]
	return res, size
}

// parseBitlistProgressive is like parseBitlist, but preserves the packed bits at
// their full ceil(size/8)-byte length instead of trimming trailing zero bytes.
// Progressive bitlists are unbounded: there is no limit to re-pad the chunk
// count back to, so trailing zero chunks are significant and must be retained
// (trimming them yields a smaller, incorrect progressive tree).
func parseBitlistProgressive(dst, buf []byte) ([]byte, uint64) {
	msb := uint8(bits.Len8(buf[len(buf)-1])) - 1
	size := uint64(8*(len(buf)-1) + int(msb))

	dst = append(dst, buf...)
	dst[len(dst)-1] &^= uint8(1 << msb) // clear the length-delimiting bit

	return dst[:(size+7)/8], size // full packed length implied by the bit count
}

// MerkleizeWithMixin is used to merkleize the last group of the hasher
func (h *Hasher) MerkleizeWithMixin(indx int, num, limit uint64) {
	h.record(opMerkleizeMixin, indx, num, limit, nil)
	input := merkleizeInput(h.buf[indx:], limit)
	// mixin with the size
	sizemix := h.tmp[:32]
	for indx := range sizemix {
		sizemix[indx] = 0
	}
	binary.LittleEndian.AppendUint64(sizemix[:0], num)
	h.buf = append(h.buf[:indx], h.doHash(input, sizemix, input)...)
}

func (h *Hasher) doHash(left, right, out []byte) []byte {
	h.hash.Write(left)
	h.hash.Write(right)
	h.hash.Sum(out[:0])
	h.hash.Reset()
	return out
}

func (h *Hasher) Merkleize(indx int) {
	h.record(opMerkleize, indx, 0, 0, nil)
	h.buf = append(h.buf[:indx], merkleizeInput(h.buf[indx:], 0)...)
}

// MerkleizeProgressive merkleizes the last group of the hasher per the spec's
// merkleize_progressive: subtrees of capacity 1, 4, 16, ... are merkleized as
// binary trees and folded right-to-left, hash(base, successor).
func (h *Hasher) MerkleizeProgressive(indx int) {
	h.record(opProgressive, indx, 0, 0, nil)
	root := merkleizeProgressiveInput(h.buf[indx:])
	h.buf = append(h.buf[:indx], root[:]...)
}

// MerkleizeProgressiveWithMixin merkleizes the last group of the hasher per
// the progressive scheme and mixes in the element count (progressive lists
// and bitlists have no limit, so the mixin carries only the length).
func (h *Hasher) MerkleizeProgressiveWithMixin(indx int, num uint64) {
	h.record(opProgressiveMixin, indx, num, 0, nil)
	root := merkleizeProgressiveInput(h.buf[indx:])
	sizemix := h.tmp[:32]
	for i := range sizemix {
		sizemix[i] = 0
	}
	binary.LittleEndian.AppendUint64(sizemix[:0], num)
	h.buf = append(h.buf[:indx], h.doHash(root[:], sizemix, root[:])...)
}

// MerkleizeProgressiveWithActiveFields merkleizes the last group of the hasher
// per the progressive scheme and mixes in a progressive container's
// active-fields bitvector: hash(root, pack_bits(active_fields)). activeFields
// is the packed bitvector (≤ 32 bytes; the chunk is zero-padded).
func (h *Hasher) MerkleizeProgressiveWithActiveFields(indx int, activeFields []byte) {
	h.record(opProgressiveActiveFields, indx, 0, 0, activeFields)
	root := merkleizeProgressiveInput(h.buf[indx:])
	aux := h.tmp[:32]
	for i := range aux {
		aux[i] = 0
	}
	copy(aux, activeFields)
	h.buf = append(h.buf[:indx], h.doHash(root[:], aux, root[:])...)
}

// PutProgressiveBitlist appends an ssz progressive bitlist (no limit; the
// progressive analog of PutBitlist).
func (h *Hasher) PutProgressiveBitlist(bb []byte) {
	var size uint64
	h.tmp, size = parseBitlistProgressive(h.tmp[:0], bb)

	indx := h.Index()
	h.AppendBytes32(h.tmp)
	h.MerkleizeProgressiveWithMixin(indx, size)
}

// merkleizeProgressiveInput chunks the buffered bytes (zero-padding the
// trailing partial chunk) and merkleizes them progressively.
func merkleizeProgressiveInput(input []byte) [32]byte {
	chunkCount := (len(input) + 31) / 32
	chunks := make([][32]byte, chunkCount)
	for i, j := 0, 0; j < chunkCount; i, j = i+32, j+1 {
		if j == chunkCount-1 {
			copy(chunks[j][:], input[i:])
		} else {
			copy(chunks[j][:], input[i:i+32])
		}
	}
	return merkleizeProgressive(chunks, 1)
}

// merkleizeProgressive implements the spec's merkleize_progressive(chunks,
// num_leaves=1): an empty input is the zero chunk; otherwise the root is
// hash(a, b) where a is the binary merkleization of the first up-to-numLeaves
// chunks (with limit numLeaves) and b is the progressive merkleization of the
// remainder with numLeaves*4. The chunks slice is consumed as scratch: the
// base subtree is capacity-clipped because merkleizeChunks appends zero-hash
// padding in place, which would otherwise clobber the successor's chunks.
func merkleizeProgressive(chunks [][32]byte, numLeaves uint64) [32]byte {
	if len(chunks) == 0 {
		return [32]byte{}
	}
	n := numLeaves
	if uint64(len(chunks)) < n {
		n = uint64(len(chunks))
	}
	a := merkleizeChunks(chunks[:n:n], numLeaves)
	b := merkleizeProgressive(chunks[n:], numLeaves*4)
	return sha256.Sum256(append(a[:], b[:]...))
}

func merkleizeInput(input []byte, limit uint64) []byte {
	chunkCount := (len(input) + 31) / 32
	chunks := make([][32]byte, chunkCount)
	for i, j := 0, 0; j < chunkCount; i, j = i+32, j+1 {
		if j == chunkCount-1 {
			copy(chunks[j][:], input[i:])
		} else {
			copy(chunks[j][:], input[i:i+32])
		}
	}

	var result [32]byte
	if limit == 0 {
		result = merkleizeChunks(chunks, uint64(chunkCount))
	} else {
		result = merkleizeChunks(chunks, limit)
	}

	return result[:]
}

// merkleizeChunks hashes a list of 32-byte elements up to chunkCount leaves.
func merkleizeChunks(elements [][32]byte, chunkCount uint64) [32]byte {
	dep := depth(chunkCount)
	// Return zerohash at depth
	if len(elements) == 0 {
		return zeroHashesRaw[dep]
	}
	for i := uint8(0); i < dep; i++ {
		layerLen := len(elements)
		oddNodeLength := layerLen%2 == 1
		if oddNodeLength {
			zerohash := zeroHashesRaw[i]
			elements = append(elements, zerohash)
		}
		outputLen := len(elements) / 2
		// gohashtree concurrently overwrites elements
		err := gohashtree.Hash(elements, elements)
		if err != nil {
			panic(err)
		}
		elements = elements[:outputLen]
	}
	return elements[0]
}

func nextPowerOfTwo(v uint64) uint {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return uint(v)
}

func getDepth(d uint64) uint8 {
	if d == 0 {
		return 0
	}
	if d == 1 {
		return 1
	}
	i := nextPowerOfTwo(d)
	return 64 - uint8(bits.LeadingZeros(i)) - 1
}

// Depth retrieves the appropriate depth for the provided trie size.
func depth(v uint64) (out uint8) {
	// bitmagic: binary search through a uint32, offset down by 1 to not round powers of 2 up.
	// Then adding 1 to it to not get the index of the first bit, but the length of the bits (depth of tree)
	// Zero is a special case, it has a 0 depth.
	// Example:
	//  (in out): (0 0), (1 0), (2 1), (3 2), (4 2), (5 3), (6 3), (7 3), (8 3), (9 4)
	if v <= 1 {
		return 0
	}
	v--
	if v&mask5 != 0 {
		v >>= bit5
		out |= bit5
	}
	if v&mask4 != 0 {
		v >>= bit4
		out |= bit4
	}
	if v&mask3 != 0 {
		v >>= bit3
		out |= bit3
	}
	if v&mask2 != 0 {
		v >>= bit2
		out |= bit2
	}
	if v&mask1 != 0 {
		v >>= bit1
		out |= bit1
	}
	if v&mask0 != 0 {
		out |= bit0
	}
	out++
	return
}

// HasherPool may be used for pooling Hashers for similarly typed SSZs.
type HasherPool struct {
	pool sync.Pool
}

// Get acquires a Hasher from the pool.
func (hh *HasherPool) Get() *Hasher {
	h := hh.pool.Get()
	if h == nil {
		return NewHasher()
	}
	return h.(*Hasher)
}

// Put releases the Hasher to the pool.
func (hh *HasherPool) Put(h *Hasher) {
	h.Reset()
	hh.pool.Put(h)
}
