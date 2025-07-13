package graphics

import (
	"fmt"

	"github.com/faideww/mc-iso/src/block"
	"github.com/faideww/mc-iso/src/region"
)

// Holds all static, shared context needed for rendering operations
type RenderContext struct {
	Assets       *block.BlockAssets
	BakedModels  map[string]BakedModel
	TextureAtlas *TextureAtlas
}

func (wc *RenderContext) GetBakedModelAt(s *region.Section, x, y, z int) (BakedModel, bool) {
	// if the coords are out of bounds, return ok=false
	if x < 0 || x > 15 || y < 0 || y > 15 || z < 0 || z > 15 {
		return BakedModel{}, false
	}

	// Look up the palette index
	paletteIndex, err := s.GetPaletteIdx(x, y, z)
	if err != nil {
		fmt.Println(err)
		return BakedModel{}, false
	}

	// fmt.Printf("palette index at %d,%d,%d=%d\n", x, y, z, paletteIndex)
	// Check for Air
	if paletteIndex == s.BlockStates.AirIdx {
		// fmt.Println(err)
		return BakedModel{}, false
	}

	blockResourceName := s.BlockStates.Palette[paletteIndex].Name

	// Map palette index to blockstate
	blockState, ok := wc.Assets.BlockStates[blockResourceName]
	if !ok {
		fmt.Printf("No blockstate definition found for resource %s\n", blockResourceName)
		return BakedModel{}, false
	}

	if blockState.Variants == nil || len(blockState.Variants) == 0 {
		fmt.Printf("Blockstate %s has no variants\n", blockResourceName)
		return BakedModel{}, false
	}

	var modelName string
	for _, variant := range blockState.Variants {
		if len(variant) > 0 {
			// TODO: for now, just pick the first variant and use it.
			modelName = variant[0].Model
			break
		}
	}

	if modelName == "" {
		fmt.Printf("Blockstate %s has no renderable variants\n", blockResourceName)
		return BakedModel{}, false
	}

	model, ok := wc.BakedModels[modelName]
	return model, ok
}
