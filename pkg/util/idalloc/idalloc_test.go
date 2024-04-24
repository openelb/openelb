package idalloc

import (
	"crypto/sha256"
	"math/bits"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestIDAllocator(t *testing.T) {
	set := map[uint32]struct{}{}

	alloc := New(192)
	for i := 1; i < 64; i++ {
		bit, err := alloc.Allocate()
		if err != nil {
			t.Errorf("Allocate error: got %v, want nil", err)
		}
		if _, ok := set[bit]; ok {
			t.Errorf("Unexpected duplicate value %d", bit)
		}
		set[bit] = struct{}{}
	}

	if uint32(len(alloc.bits)) != initialSize/bits.UintSize {
		t.Errorf("Unexpected length for bits, got %d want %d", len(alloc.bits), initialSize/bits.UintSize)
	}

	for i, v := range alloc.bits {
		if v != ^uint(0) {
			t.Errorf("Unexpected bits in block %d, got %d want %d", i, v, ^uint(0))
		}
	}

	alloc.Free(62)
	bit, err := alloc.Allocate()
	if err != nil {
		t.Errorf("Allocate error: got %v, want nil", err)
	}
	if bit != 62 {
		t.Errorf("Unexpected allocation: got %d, want %d", bit, 62)
	}

	bit, err = alloc.Allocate()
	if err != nil {
		t.Errorf("Allocate error: got %v, want nil", err)
	}
	if bit != 64 {
		t.Errorf("Unexpected allocation: got %d, want %d", bit, 64)
	}

	left := alloc.maxBlocks*bits.UintSize - (bit + 1)
	for i := uint32(0); i < left; i++ {
		bit, err := alloc.Allocate()
		if err != nil {
			t.Errorf("Allocate error: got %v, want nil", err)
		}
		if _, ok := set[bit]; ok {
			t.Errorf("Unexpected duplicate value %d", bit)
		}
		set[bit] = struct{}{}
	}

	for i, v := range alloc.bits {
		if v != ^uint(0) {
			t.Errorf("Unexpected bits in block %d, got %d, want %d", i, v, ^uint(0))
		}
	}

	bit, err = alloc.Allocate()
	if bit != IDMaxLimit {
		t.Errorf("Unexpected value for bit, got %d, want %d", bit, IDMaxLimit)
	}
	if err != ErrIDsExhausted {
		t.Errorf("Unexpected value for err, got %v, want %v", err, ErrIDsExhausted)
	}

	p := func() (result interface{}) {
		defer func() {
			result = recover()
		}()
		alloc.Free(512)
		return result
	}
	if p == nil {
		t.Errorf("Expected panic but did not")
	}
}

func TestBadMaxValue(t *testing.T) {
	p := func() (result interface{}) {
		defer func() {
			result = recover()
		}()
		New(0)
		return result
	}()
	if p == nil {
		t.Errorf("Expected panic but did not")
	}

	p = func() (result interface{}) {
		defer func() {
			result = recover()
		}()
		New(127)
		return result
	}()
	if p == nil {
		t.Errorf("Expected panic but did not")
	}
}

func TestAllocateWithHash(t *testing.T) {
	set := map[uint32]struct{}{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	alloc := New(256)
	for i := 0; i < 255; i++ {
		bytes := sha256.Sum256([]byte("node" + strconv.Itoa(r.Intn(30000))))
		bit, err := alloc.AllocateWithHash(bytes)
		if err != nil {
			t.Errorf("Allocate error: got %v, want nil", err)
		}
		if _, ok := set[bit]; ok {
			t.Errorf("Unexpected duplicate value %d", bit)
		}
		set[bit] = struct{}{}
	}

	for i, v := range alloc.bits {
		if v != ^uint(0) {
			t.Errorf("Unexpected bits in block %d, got %d want %d", i, v, ^uint(0))
		}
	}
}
