package graphics

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/faideww/mc-iso/src/block"
	"github.com/faideww/mc-iso/src/region"
	"github.com/faideww/mc-iso/src/util"
)

// Bundles world-absolute and section-local coordinates
// together so that they can be passed around together
type BlockPositionContext struct {
	// Absolute world coordinates (used for PRNG)
	World util.IntVector3

	// Section-local coordinates (used for palette lookups and neighbor checks)
	Local util.IntVector3
}

// Holds all static, shared context needed for rendering operations
type RenderContext struct {
	Assets           *block.BlockAssets
	BakedBlockStates map[string]BakedBlockState
	TextureAtlas     *TextureAtlas
	Random           util.RandomSource
}

func (wc *RenderContext) ResolveVariantBlockModel(s *region.Section, pos *BlockPositionContext) (BakedVariantModel, bool) {
	// if the coords are out of bounds, return ok=false
	if pos.Local.X < 0 || pos.Local.X > 15 || pos.Local.Y < 0 || pos.Local.Y > 15 || pos.Local.Z < 0 || pos.Local.Z > 15 {
		return BakedVariantModel{}, false
	}

	// Look up the palette index
	paletteIndex, err := s.GetPaletteIdx(pos.Local.X, pos.Local.Y, pos.Local.Z)
	if err != nil {
		fmt.Println(err)
		return BakedVariantModel{}, false
	}

	// Check for Air
	// TODO: is it possible we want to actually handle air as a blockstate? even though it doesn't have a model?
	if paletteIndex == s.BlockStates.AirIdx {
		// fmt.Println(err)
		return BakedVariantModel{}, false
	}

	blockWorldState := s.BlockStates.Palette[paletteIndex]

	bakedBlockState, blockState := wc.BakedBlockStates[blockWorldState.Name]
	if !blockState {
		fmt.Printf("block %d%d%d (%s) has no baked blockstate\n", pos.Local.X, pos.Local.Y, pos.Local.Z, blockWorldState.Name)
		return BakedVariantModel{}, blockState
	}

	// Since the blockstate properties from the paletteare unmarshaled into a
	// map, we have no guarantee of their order. We're assuming that blockstate
	// variants always have their properties sorted alphabetically, and are
	// always fully enumerated.
	// TODO: validate this assumption, and maybe add some fallback code to search
	// for sparsely-defined blockstate properties?
	type VariantProperty struct {
		Key   string
		Value string
	}
	properties := []VariantProperty{}
	for propName, propValue := range blockWorldState.Properties {
		properties = append(properties, VariantProperty{propName, propValue})
	}
	slices.SortFunc(properties, func(a, b VariantProperty) int {
		return cmp.Compare(a.Key, b.Key)
	})

	var b strings.Builder
	for _, property := range properties {
		fmt.Fprintf(&b, "%s=%s", property.Key, property.Value)
	}
	propString := b.String()

	if bakedBlockState.IsMultipart {
		// TODO: handle multipart
		fmt.Printf("block %d,%d,%d (%s) is multipart; not implemented\n", pos.Local.X, pos.Local.Y, pos.Local.Z, blockWorldState.Name)
		return BakedVariantModel{}, false
	} else {
		variant, variantOk := bakedBlockState.Variants[propString]
		if !variantOk {
			// Fall back to default variant if possible
			variant, variantOk = bakedBlockState.Variants[""]
			if !variantOk {
				return BakedVariantModel{}, variantOk
			}
		}

		if variant.IsWeightedChoice {
			// Use the PRNG to select a variant

			// PRNG seeding uses the world coordinates
			seed := util.GetSeed(pos.World.X, pos.World.Y, pos.World.Z)

			// Seed the PRNG with the world coordinates
			wc.Random.SetSeed(seed)

			// Yield a random int64
			randomLong := wc.Random.NextLong()
			if randomLong < 0 {
				randomLong *= -1
			}
			randomIndex := randomLong % int64(variant.TotalWeight)

			// Find the selected variant by looping over each and subtracting its
			// weight from the index until we reach 0
			for _, vm := range variant.Models {
				randomIndex -= int64(vm.Weight)
				if randomIndex < 0 {
					return vm, true
				}
			}

			// We can't find a variant, for whatever reason
			return BakedVariantModel{}, false
		} else {
			return variant.Models[0], true
		}
	}
}
