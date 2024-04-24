package idalloc

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
)

// IDMaxLimit is the maximum possible ID value that could be returned by an
// IDAllocator.
const IDMaxLimit uint32 = math.MaxUint32

// ErrIDsExhausted is returned when an IDAllocator is unable to fulfill an
// allocation request.
var ErrIDsExhausted = errors.New("no more IDs available")

// IDAllocator is an allocator for ID values. It is implemented using a bitmap
// that is grown as necessary.
type IDAllocator struct {
	bits      []uint
	maxBlocks uint32
}

const initialSize = uint32(64)

// New creates a new IDAllocator that may allocate up to numIDs values. numIDs
// must be a multiple of 64.
func New(numIDs uint32) IDAllocator {
	// For this check we use initialSize (64) instead of bits.UintSize so that we
	// can be consistent between CPU architectures.
	if numIDs == 0 || numIDs%initialSize != 0 {
		panic(fmt.Sprintf("numIDs must be non-zero and divisible by %d", initialSize))
	}

	numBlocks := (initialSize + bits.UintSize - 1) / bits.UintSize
	allocator := IDAllocator{
		bits:      make([]uint, numBlocks),
		maxBlocks: (numIDs + bits.UintSize - 1) / bits.UintSize,
	}

	// Preallocate ID 0
	allocator.bits[0] |= 1

	return allocator
}

// Allocate finds an unused ID, sets it as used, and returns its value.
// If the IDAllocator is full and there are no more IDs available, id
// will be set to IDMaxLimit and err will be set to ErrIDsExhausted.
func (a *IDAllocator) Allocate() (id uint32, err error) {
	i := uint32(0)
	for {
		curBlock := a.bits[i]
		if curBlock != ^uint(0) {
			bb := uint32(bits.TrailingZeros(^curBlock))
			a.bits[i] = curBlock | (uint(1) << bb)
			return uint32(i*bits.UintSize + bb), nil
		}

		i++
		if i == uint32(len(a.bits)) && !a.grow() {
			return IDMaxLimit, ErrIDsExhausted
		}
	}
}

// AllocateWithHash allocates an ID based on the given SHA256 hash. The same hash
// will always result in the same ID. If the calculated ID is already allocated,
// it will try the next ID until an unallocated one is found.
func (a *IDAllocator) AllocateWithHash(hash [sha256.Size]byte) (id uint32, err error) {
	// Convert the first 4 bytes of the hash to an uint32
	initialID := binary.BigEndian.Uint32(hash[:4]) % (a.maxBlocks * bits.UintSize)

	for i := uint32(0); i < a.maxBlocks*bits.UintSize; i++ {
		id = (initialID + i) % (a.maxBlocks * bits.UintSize)
		i, mask := id/bits.UintSize, uint(1)<<(id%bits.UintSize)

		for {
			if i >= uint32(len(a.bits)) {
				if !a.grow() {
					return IDMaxLimit, ErrIDsExhausted
				}
			} else {
				break
			}
		}

		if a.bits[i]&mask == 0 {
			a.bits[i] |= mask
			return id, nil
		}
	}

	return IDMaxLimit, ErrIDsExhausted
}

// Free marks id as unused. id must have been previously returned by a
// successful call to Allocate.
func (a *IDAllocator) Free(id uint32) {
	i, mask := id/bits.UintSize, uint(1)<<(id%bits.UintSize)
	a.bits[i] &= ^mask
}

func (a *IDAllocator) grow() bool {
	n, m := uint32(len(a.bits)), a.maxBlocks
	if n >= m {
		return false
	}

	// Try to double the size, but if that would exceed our maximum then just
	// allocate up to the max.
	if 2*n > m {
		n = m - n
	}

	a.bits = append(a.bits, make([]uint, n)...)
	return true
}
