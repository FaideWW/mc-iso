package region

import (
	"errors"
)

type Chunk struct {
	Loaded      bool
	DataVersion int       `nbt:"DataVersion"`
	XPos        int32     `nbt:"xPos"`
	ZPos        int32     `nbt:"zPos"`
	YPos        int32     `nbt:"yPos"`
	Status      string    `nbt:"Status"`
	LastUpdate  int64     `nbt:"LastUpdate"`
	Sections    []Section `nbt:"sections"`
}

type Section struct {
	Y           int8                 `nbt:"Y"`
	BlockStates Palette[PaletteData] `nbt:"block_states"`
	Biomes      Palette[string]      `nbt:"biomes"`
	// BlockLight  [2048]byte   `nbt:"BlockLight"`
	// SkyLight    [2048]byte   `nbt:"SkyLight"`
}

type Palette[T any] struct {
	Palette []T     `nbt:"palette"`
	Data    []int64 `nbt:"data"`

	indexSize int // cached value of the size of the palette index (see Index())
}

type PaletteData struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties"`
}

//
// We have two options:
// 1. re-pack the data so that we have the correct # of entries and they point to the correct block when we index them
// 2.  create a getter method which can do this work on the fly.

type PaletteIndices interface {
	Index(i int) int64
}

const (
	BLOCK_PALETTE_SIZE = 4096
	BIOME_PALETTE_SIZE = 64
)

// Returns the palette entry at index i, or an error if i is out of bounds
// Palette indices are packed in such a way that they are only as large as they
// need to be to store the entire palette. eg. if the palette has 15 entries,
// the indices will be 4 bits wide. if the palette has 17 entries, the indices
// will be 5 bits wide, and so on.
func (p Palette[T]) Index(i int, useNewPacking bool) (int64, error) {
	if len(p.Data) == 0 {
		return 0, nil
	}

	// memoize the index size so we don't have to keep re-calculating it for each index
	var indexSize int
	if p.indexSize == 0 {
		paletteSize := len(p.Palette)
		indexSize = bitSize(paletteSize - 1)
		p.indexSize = indexSize
	} else {
		indexSize = p.indexSize
	}

	// as of MC 1.16, these entries are aligned to the int64 boundaries; meaning
	// that they will only pack into one int64 as many full indexes as will fit,
	// or floor(64/indexSize).
	// prior to 1.16 (DataVersion 2556?) they were packed across multiple
	// elements, so we will need a different unpacking scheme for these values (TODO)
	if !useNewPacking {
		return -1, errors.New("Old palette packing scheme not yet supported")
	}

	bitIndex := i * indexSize

	// find the element in the long data we need to look at, and the offset into
	// that entry
	longDataIndex := bitIndex / 64
	longDataOffset := (bitIndex % 64) / indexSize

	// to extract the index from the long data, which is stored bit-wise as
	// longBits[n:n+indexSize], we right-shift the entry to drop all the bits to
	// the right, and then bitwise-AND with a masking value to drop bits to the
	// left
	rshift := 64 - (longDataOffset + indexSize)
	mask := IntPow(2, indexSize) - 1

	result := (p.Data[longDataIndex] >> int64(rshift)) & int64(mask)

	return result, nil
}

// Given an integer i, returns the smallest number of bits that can represent i.
func bitSize(i int) int {
	// increment exp until 2^exp is greater than i
	exp := 1

	for IntPow(2, exp) <= i {
		exp++
	}

	return exp
}

// Raise n to the mth power, where all inputs and outputs are ints
func IntPow(n, m int) int {
	if m == 0 {
		return 1
	}
	if m == 1 {
		return n
	}

	result := n
	for i := 2; i <= m; i++ {
		result *= n
	}
	return result
}
