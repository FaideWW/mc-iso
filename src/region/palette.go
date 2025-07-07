package region

import (
	"errors"
)

type Palette[T any] struct {
	Palette []T     `nbt:"palette"`
	Data    []int64 `nbt:"data"`

	AirIdx    int64 // palette index of "air" blocks (or -1 if none are present)
	IndexSize int   // size of the palette index (see Index())
}

type PaletteData struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties"`
}

//
// We have two options:
// 1. re-pack the data so that we have the correct # of entries and they point to the correct block when we index them
// 2.  create a getter method which can do this work on the fly.
// TODO: we probably want to unpack the palette indices so that we don't have to do any computation to retrieve them later

type PaletteIndices interface {
	Index(i int) int64
}

const (
	BLOCK_PALETTE_SIZE = 4096
	BIOME_PALETTE_SIZE = 64
)

func (p Palette[T]) GetIndexSize() int {
	paletteSize := len(p.Palette)
	indexSize := bitSize(paletteSize - 1)
	// Indices have a minimum size of 4 bits
	if indexSize < 4 {
		indexSize = 4
	}
	return indexSize
}

// Returns the palette entry at index i, or an error if i is out of bounds
// Palette indices are packed in such a way that they are only as large as they
// need to be to store the entire palette. eg. if the palette has 15 entries,
// the indices will be 4 bits wide (2^4=16). if the palette has 17 entries, the indices
// will be 5 bits wide (2^5=32), and so on.
func (p Palette[T]) Index(i int, useNewPacking bool) (int64, error) {
	if len(p.Data) == 0 {
		return 0, nil
	}

	// as of MC 1.16, these entries are aligned to the int64 boundaries; meaning
	// that they will only pack into one int64 as many full indexes as will fit,
	// or floor(64/indexSize).
	// TODO: prior to 1.16 (DataVersion 2556?) they were packed across multiple
	// elements, so we will need a different unpacking scheme for these values
	if !useNewPacking {
		return -1, errors.New("Old palette packing scheme not yet supported")
	}

	// First, we need to find how many indices can fit into each array element.
	indicesPerElement := 64 / p.IndexSize

	// Then, we need to compute which element will contain the index we care
	// about, and the offset (in bits) into that element where our index is
	longDataIndex := i / indicesPerElement
	longDataOffset := (i % indicesPerElement) * p.IndexSize

	// fmt.Printf("index: %d (%d bits)\n", i, indexStart)
	// fmt.Printf("longDataIndex: %d\n", longDataIndex)
	// fmt.Printf("longDataOffset: %d\n", longDataOffset)

	// fmt.Printf("%64b\n", p.Data[longDataIndex])
	// for i := 63; i >= longDataOffset+p.IndexSize; i-- {
	// 	fmt.Printf(" ")
	// }
	// for i := 0; i < p.IndexSize; i++ {
	// 	fmt.Printf("^")
	// }
	// fmt.Println()

	// ----- OLD CODE

	// // find the element in the long data we need to look at, and the offset into
	// // that entry
	// longDataIndex := bitIndex / 64
	// longDataOffset := (bitIndex % 64) / p.IndexSize
	// ----- END OLD CODE

	// to extract the index from the long data, which is stored bit-wise as
	// longBits[n:n+indexSize], we right-shift the entry to drop all the bits to
	// the right, and then bitwise-AND with a masking value to drop bits to the
	// left
	rshift := 64 - (longDataOffset + p.IndexSize)
	mask := IntPow(2, p.IndexSize) - 1

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
