package region

import (
	// "fmt"
	"log"
)

type Section struct {
	Y           int8                 `nbt:"Y"`
	BlockStates Palette[PaletteData] `nbt:"block_states"`
	Biomes      Palette[string]      `nbt:"biomes"`
	// BlockLight  [2048]byte   `nbt:"BlockLight"`
	// SkyLight    [2048]byte   `nbt:"SkyLight"`
}

// Tests whether a block is air.
func (s *Section) IsAir(x, y, z int) bool {
	if x < 0 || x > 15 || y < 0 || y > 15 || z < 0 || z > 15 {
		return true
	}

	if s.BlockStates.AirIdx == -1 {
		return false
	}

	// Blocks are ordered YZX, for compression purposes
	blockIndex := y*16*16 + z*16 + x

	paletteIndex, err := s.BlockStates.Index(blockIndex, true)
	if err != nil {
		log.Printf("PALETTE ERROR: %s\n", err)
		return false
	}

	return paletteIndex == s.BlockStates.AirIdx
}

func (s *Section) GetPaletteIdx(x, y, z int) (int64, error) {
	// Blocks are ordered YZX, for compression purposes
	blockIndex := y*16*16 + z*16 + x
	idx, err := s.BlockStates.Index(blockIndex, true)
	// fmt.Printf("(%d) ", idx)
	return idx, err
}

// Tests whether a block can't be seen through (ie. a block behind it cannot be seen from in front, or can't be affected by a light source placed in front)
// TODO: currently the only test supported is for whether the block is air. Add support for others (glass, sparse models, etc.)
func (s *Section) IsOpaque(x, y, z int) bool {

	isAir := s.IsAir(x, y, z)

	return !isAir
}
