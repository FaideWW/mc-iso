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

	bitpackSize int // cached value of the size of the palette index (see Index())
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
//
// (additional note: as of MC 1.16, these entries are aligned to the int64
// boundaries; meaning that they will only pack into one int64 as many full
// indexes as will fit, or floor(64/indexSize). prior to 1.16 (DataVersion
// 2556?) they were packed across multiple elements, presumably)
func (p Palette[T]) Index(i int, useNewPacking bool) (int64, error) {
	if len(p.Data) == 0 {
		return 0, nil
	}
	var bitpackSize int
	if p.bitpackSize == 0 {
		paletteSize := len(p.Palette)
		bitpackSize = bitSize(paletteSize - 1)
		p.bitpackSize = bitpackSize
	} else {
		bitpackSize = p.bitpackSize
	}

	// fmt.Printf("index size: %d\n", bitpackSize)

	if !useNewPacking {
		return -1, errors.New("Old palette packing scheme not yet supported")
	}

	bitIndex := i * bitpackSize

	longDataIndex := bitIndex / 64
	longDataOffset := (bitIndex % 64) / bitpackSize

	// fmt.Printf("long index: %d (%d)\n", longDataIndex, p.Data[longDataIndex])
	// fmt.Printf("long offset: %d\n", longDataOffset)
	// bit hacking time
	rshift := 64 - (longDataOffset + bitpackSize)
	mask := IntPow(2, bitpackSize) - 1

	// fmt.Printf("shift right by %d bits - mask: %d\n", rshift, mask)
	result := (p.Data[longDataIndex] >> int64(rshift)) & int64(mask)

	// fmt.Printf("result: %d\n", result)
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
