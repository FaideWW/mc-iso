package graphics

import (
	"errors"
	"fmt"
	"math"

	"github.com/faideww/mc-iso/src/block"
	rl "github.com/gen2brain/raylib-go/raylib"
)

/**

Interior face-culling
---

Adjacent blocks that are flush up against each other do not need their touching faces rendered. This can significantly reduce the number of triangles drawn each frame, as most blocks are covered on one, some, or all sides.

During the baking stage, we pre-compute the data needed to determine if a face can be culled when building our world mesh. This consists of two components:

1. The face-to-be-culled's "Cullface" property. This specifies a direction to check for a neighboring block whose opposite face is capable of culling (described next).
2. The block's "FaceCullingMask". This describes which faces of the block are solid, opaque, and flush to the block grid. In other words, will this face completely obscure the face of the block next to it?

These two properties can be checked in the rendering stage to determine whether to draw the face or not.

*/

type FaceDir int

const (
	FaceWest = iota
	FaceEast
	FaceNorth
	FaceSouth
	FaceDown
	FaceUp
)

type BakedFace struct {
	Vertices [4]rl.Vector3 // pre-rotated, model-space coordinates [0-1]
	Uv       [4]rl.Vector2 // Texture atlas uv
	Normal   rl.Vector3    // Normal vector of the face, used for lighting
	Cullface FaceDir       // Which direction to test whether this face should be culled
}

type BakedModel struct {
	Faces []BakedFace

	RenderType      RenderType
	FaceCullingMask uint8 // Which faces of this block are capable of culling?
}

type BakedVariantModel struct {
	Model *BakedModel

	ModelName string
	Weight    int
	RotationX int
	RotationY int
	Uvlock    bool
}

type BakedVariant struct {
	Models           []BakedVariantModel
	IsWeightedChoice bool
	TotalWeight      int
}

type BakedMultipartCase struct {
	Condition block.MultipartCondition
	Model     BakedVariant
}

type BakedBlockState struct {
	// For blocks using "variants", we store each variant keyed by its property conditions (eg. "axis=y")
	Variants map[string]BakedVariant
	// For blocks using "multipart", we store the raw rules directly, as they must be evaluated at runtime
	Multipart   []BakedMultipartCase
	IsMultipart bool
}

type BakedBlockStateMap map[string]BakedBlockState

// type ModelMap map[string]BakedModel

var bakedModelCache map[string]BakedModel

// Two lists of blocks that are geometrically full 16x16x16 cubes, but must be
// treated as non-solid by the renderer (more specifically, the face culling
// logic) due to hard-coded transparent or cutout behavior in the game client.
// Entries in these lists will override the result of our geometric tests for
// FaceCullingMask. This list isn't complete or comprehensive, but covers the
// most common cases so we can avoid the more expensive translucency test for
// the known blocks.
// Refer to https://minecraft.wiki/w/Opacity for a complete list

// var RenderTypeExceptionsTranslucent = []string{}
var RenderTypeExceptionsTranslucent = []string{
	// Full-block translucent types
	"minecraft:block/barrier",
	"minecraft:block/beacon",
	"minecraft:block/frosted_ice",
	"minecraft:block/water",
	"minecraft:block/ice",
	"minecraft:block/honey_block", // Consists of an inner, opaque block, and an outer, transparent block
	"minecraft:block/slime_block", // Consists of an inner, opaque block, and an outer, transparent block

	// Glass variants (annoyingly, these are all separately defined and don't inherit from a common glass model)
	"minecraft:block/tinted_glass",
	"minecraft:block/white_stained_glass",
	"minecraft:block/orange_stained_glass",
	"minecraft:block/magenta_stained_glass",
	"minecraft:block/light_blue_stained_glass",
	"minecraft:block/yellow_stained_glass",
	"minecraft:block/lime_stained_glass",
	"minecraft:block/pink_stained_glass",
	"minecraft:block/gray_stained_glass",
	"minecraft:block/light_gray_stained_glass",
	"minecraft:block/cyan_stained_glass",
	"minecraft:block/purple_stained_glass",
	"minecraft:block/blue_stained_glass",
	"minecraft:block/brown_stained_glass",
	"minecraft:block/green_stained_glass",
	"minecraft:block/red_stained_glass",
	"minecraft:block/black_stained_glass",
}

// var RenderTypeExceptionsCutout = []string{}
var RenderTypeExceptionsCutout = []string{
	// Full-block cutout types
	"minecraft:block/glass",
	"minecraft:block/copper_grate",
	"minecraft:block/mangrove_roots",
	"minecraft:block/spawner",

	// Leaves variants (only transparent in "fancy" graphics mode)
	"minecraft:block/leaves",
	"minecraft:block/oak_leaves",
	"minecraft:block/spruce_leaves",
	"minecraft:block/birch_leaves",
	"minecraft:block/junlge_leaves",
	"minecraft:block/acaia_leaves",
	"minecraft:block/dark_oak_leaves",
	"minecraft:block/mangrove_leaves",
	"minecraft:block/cherry_leaves",
	"minecraft:block/pale_oak_leaves",
	"minecraft:block/azalea_leaves",
	"minecraft:block/flowering_azalea_leaves",
}

var faceIndices = [...][4]int{
	{0, 4, 6, 2}, // -X
	{5, 1, 3, 7}, // +X
	{1, 0, 2, 3}, // -Z
	{4, 5, 7, 6}, // +Z
	{0, 1, 5, 4}, // -Y
	{6, 7, 3, 2}, // +Y
}

var faceNormals = [...]rl.Vector3{
	{X: -1, Y: 0, Z: 0}, // -X
	{X: 1, Y: 0, Z: 0},  // +X
	{X: 0, Y: 0, Z: -1}, // -Z
	{X: 0, Y: 0, Z: 1},  // +Z
	{X: 0, Y: -1, Z: 0}, // -Y
	{X: 0, Y: 1, Z: 0},  // +Y
}

// Opposing faces are indexed adjacently: 0=-X, 1=+X, 2=-Z, 3=+Z, etc.
func getOpposingFace(face FaceDir) FaceDir {
	if face%2 == 1 {
		return face - 1
	}

	return face + 1
}

func faceDirectionToIndex(dir string) FaceDir {
	var idx FaceDir
	switch dir {
	case "west": // -X
		{
			idx = FaceWest
		}
	case "east": // +X
		{
			idx = FaceEast
		}
	case "north": // -Z
		{
			idx = FaceNorth
		}
	case "south": // +Z
		{
			idx = FaceSouth
		}
	case "down": // -Y
		{
			idx = FaceDown
		}
	case "up": // +Y
		{
			idx = FaceUp
		}
	}
	return idx
}

func getModelDescription(models map[string]block.BlockModel, modelId string) (block.BlockModel, error) {
	var resolvedDesc block.BlockModel
	resolvedDesc.Textures = make(map[string]string)
	modelDesc, ok := models[modelId]
	if !ok {
		return resolvedDesc, fmt.Errorf("no block model called %s found", modelId)
	}

	// Grab any properties present on the parent (recursively)
	if modelDesc.Parent != "" {
		parentDesc, err := getModelDescription(models, modelDesc.Parent)
		if err != nil {
			return modelDesc, errors.Join(err, fmt.Errorf("block model %s parent error", modelId))
		}
		// populate the texture variables map.
		for key, texVar := range parentDesc.Textures {
			resolvedDesc.Textures[key] = texVar
		}
		resolvedDesc.Elements = parentDesc.Elements
	}

	// If the model has elements, they overwrite all elements from the parent
	if len(modelDesc.Elements) > 0 {
		resolvedDesc.Elements = modelDesc.Elements
	}

	// If the model has textures, they are gracefully merged with the parent's
	// textures (texture variables with the same name are overwritten by the
	// child model)
	for key, texVar := range modelDesc.Textures {
		resolvedDesc.Textures[key] = texVar
	}

	return resolvedDesc, nil
}

func resolveTextureVariable(initialKey string, textures map[string]string) (string, bool) {
	currentKey := initialKey

	// Safeguard to prevent us from looping infinitely in case of circular references
	for i := 0; i < 10; i++ {
		if len(currentKey) > 0 && currentKey[0] == '#' {
			nextKey, ok := textures[currentKey[1:]]
			if !ok {
				return "", false
			}

			currentKey = nextKey
		} else {
			return currentKey, true
		}
	}
	fmt.Printf("[baker] Warning: texture resolution reached the maximum lookup iterations for key '%s'; check for circular references.\n", initialKey)
	return "", false
}

func bakeModel(atlas *TextureAtlas, models map[string]block.BlockModel, modelId string) (BakedModel, error) {
	if cachedModel, ok := bakedModelCache[modelId]; ok {
		return cachedModel, nil
	}
	var model BakedModel

	modelDesc, err := getModelDescription(models, modelId)
	if err != nil {
		return model, errors.Join(fmt.Errorf("error getting model description"), err)
	}

	var faceCullingMask uint8 = 0

	isOnRenderExceptionList := false

	// Render type starts as OPAQUE, and based on various criteria can be demoted
	// to CUTOUT and then to TRANSLUCENT
	resolvedRenderType := RTOpaque
	for _, exceptionId := range RenderTypeExceptionsCutout {
		if modelId == exceptionId {
			isOnRenderExceptionList = true
			resolvedRenderType = RTCutout
			break
		}
	}

	for _, exceptionId := range RenderTypeExceptionsTranslucent {
		if modelId == exceptionId {
			isOnRenderExceptionList = true
			resolvedRenderType = RTTranslucent
			break
		}
	}

	// NOTE: if `elements` is defined, it overwrites the parent model's elements entirely.
	for _, element := range modelDesc.Elements {

		from := rl.Vector3{
			X: rl.Clamp(element.From.X, -16, 32),
			Y: rl.Clamp(element.From.Y, -16, 32),
			Z: rl.Clamp(element.From.Z, -16, 32),
		}
		to := rl.Vector3{
			X: rl.Clamp(element.To.X, -16, 32),
			Y: rl.Clamp(element.To.Y, -16, 32),
			Z: rl.Clamp(element.To.Z, -16, 32),
		}

		minP, maxP := sortCuboidPoints(from, to)
		// scale the cube down to world-space
		minP = rl.Vector3Scale(minP, 1.0/16)
		maxP = rl.Vector3Scale(maxP, 1.0/16)

		// fmt.Printf("Baking element with bounds: minP=%+v, maxP=%+v\n", minP, maxP)
		// fmt.Printf("Original bounds: from=%+v, to=%+v\n", from, to)

		vertices := [8]rl.Vector3{
			{X: minP.X, Y: minP.Y, Z: minP.Z},
			{X: maxP.X, Y: minP.Y, Z: minP.Z},
			{X: minP.X, Y: maxP.Y, Z: minP.Z},
			{X: maxP.X, Y: maxP.Y, Z: minP.Z},
			{X: minP.X, Y: minP.Y, Z: maxP.Z},
			{X: maxP.X, Y: minP.Y, Z: maxP.Z},
			{X: minP.X, Y: maxP.Y, Z: maxP.Z},
			{X: maxP.X, Y: maxP.Y, Z: maxP.Z},
		}

		fmt.Printf("[baker] model %s examining texture variables\n", modelId)
		fmt.Printf("model textures=%s\n", modelDesc.Textures)
		for faceDir, faceDesc := range element.Faces {

			// First, try to resolve the face's declared texture
			texId, texOk := resolveTextureVariable(faceDesc.Texture, modelDesc.Textures)
			if !texOk {
				// If we can't find it, try to fall back to #all instead
				texId, texOk = resolveTextureVariable("#all", modelDesc.Textures)
			}
			fmt.Printf("texture key=%s (resolved=%s)\n", faceDesc.Texture, texId)

			var uvRect rl.Rectangle
			if texOk {
				var ok bool
				// Look up the texcoords in the atlas
				uvRect, ok = atlas.UVMap[texId]
				if !ok {
					texOk = false
				}
			}
			if !texOk {
				// If texture variable resolution or texture lookup fails, fallback to
				// the "missing texture" texture, found at the top left of the atlas
				uvRect = rl.Rectangle{
					X:      0,
					Y:      0,
					Width:  float32(atlas.TileSize) / float32(atlas.Atlas.Width),
					Height: float32(atlas.TileSize) / float32(atlas.Atlas.Height),
				}

			}

			if faceDesc.Uv.X1 != 0 || faceDesc.Uv.X2 != 0 || faceDesc.Uv.Y1 != 0 || faceDesc.Uv.Y2 != 0 {
				// If the face describes its own uv, map those into texture atlas space and use those instead of the default uv rect.
				faceUvRect := rl.Rectangle{
					X:      float32(faceDesc.Uv.X1) / 16.0,
					Y:      float32(faceDesc.Uv.Y1) / 16.0,
					Width:  float32(faceDesc.Uv.X2-faceDesc.Uv.X1) / 16.0,
					Height: float32(faceDesc.Uv.Y2-faceDesc.Uv.Y1) / 16.0,
				}

				uvRect.X = uvRect.X + (faceUvRect.X * uvRect.Width)
				uvRect.Y = uvRect.Y + (faceUvRect.Y * uvRect.Height)
				uvRect.Width = uvRect.Width * faceUvRect.Width
				uvRect.Height = uvRect.Height * faceUvRect.Height
			}

			faceIdx := faceDirectionToIndex(faceDir)
			var quadVerts [4]rl.Vector3

			// convert the uv rectangle into vertices which are in the
			// same order as the vertices of the face quad.
			uvVerts := [4]rl.Vector2{
				{X: uvRect.X, Y: uvRect.Y + uvRect.Height},                // (u0, v1) lower-left
				{X: uvRect.X + uvRect.Width, Y: uvRect.Y + uvRect.Height}, // (u1, v1) lower-right
				{X: uvRect.X + uvRect.Width, Y: uvRect.Y},                 // (u1, v0) upper-right
				{X: uvRect.X, Y: uvRect.Y},                                // (u0, v0) upper-left
			}

			// pick the indices for the face we're looking at from the table, and
			// map them to the verts we derived earlier
			indices := faceIndices[faceIdx]
			for i := 0; i < 4; i++ {
				quadVerts[i] = vertices[indices[i]]
			}

			// fmt.Printf("[baker] faceDir=%s faceIdx=%d indices=%+v quadVerts=%+v\n", faceDir, faceIdx, indices, quadVerts)

			// Render type texture sampling. We need to test the alpha
			// channel of the face's texture rect, in order to determine whether
			// the block's render type should be demoted from OPAQUE to CUTOUT or
			// TRANSLUCENT.
			// - If the block is OPAQUE and any texel has an alpha value of 0, the
			//   block is demoted to CUTOUT
			// - If the block is OPAQUE or CUTOUT and any texel has an alpha value
			//   between 0 and 255, the block is demoted to TRANSLUCENT

			if !isOnRenderExceptionList && resolvedRenderType != RTTranslucent {
				computedRenderType := computeTextureRenderType(faceDesc.Uv, atlas.imageDataCache[texId])
				// RenderTypes are defined in order from most strict to least strict. If we compute a "higher" (less strict) render type than the block's current render type, the block is demoted to that computed type.
				if computedRenderType > resolvedRenderType {
					resolvedRenderType = computedRenderType
				}
			}

			// Face culling geometric test. Each face is tested for whether it
			// completely covers the block opposite of it. If so, the face is added
			// to the culling mask.
			// Conditions:
			// 1. Model must be OPAQUE
			// 2. Element must be unrotated
			// 3. Element's minP and maxP must completely cover the side in question
			// 4. Element must have a face on that side
			if resolvedRenderType == RTOpaque && element.Rotation.Angle == 0 {
				faceFullyCovers := false
				switch faceIdx {
				case 0: // -X
					if minP.X == 0 && minP.Y <= 0 && maxP.Y >= 16 && minP.Z <= 0 && maxP.Z >= 16 {
						faceFullyCovers = true
					}
				case 1: // +X
					if maxP.X == 16 && minP.Y <= 0 && maxP.Y >= 16 && minP.Z <= 0 && maxP.Z >= 16 {
						faceFullyCovers = true
					}
				case 2: // -Z
					if minP.Z == 0 && minP.Y <= 0 && maxP.Y >= 16 && minP.X <= 0 && maxP.X >= 16 {
						faceFullyCovers = true
					}
				case 3: // +Z
					if maxP.Z == 16 && minP.Y <= 0 && maxP.Y >= 16 && minP.X <= 0 && maxP.X >= 16 {
						faceFullyCovers = true
					}
				case 4: // -Y
					if minP.Y == 0 && minP.Y <= 0 && maxP.Y >= 16 && minP.Z <= 0 && maxP.Z >= 16 {
						faceFullyCovers = true
					}
				case 5: // +Y
					if maxP.Y == 16 && minP.Y <= 0 && maxP.Y >= 16 && minP.Z <= 0 && maxP.Z >= 16 {
						faceFullyCovers = true
					}
				}
				if faceFullyCovers {
					faceCullingMask |= (1 << faceIdx)
				}
			}

			normal := faceNormals[faceDirectionToIndex(faceDir)]

			quadVerts, normal = rotateFace(quadVerts, normal, element.Rotation)

			bakedFace := BakedFace{
				Vertices: quadVerts,
				Uv:       uvVerts,
			}
			bakedFace.Normal = normal
			bakedFace.Cullface = -1
			if faceDesc.Cullface != "" {
				bakedFace.Cullface = faceDirectionToIndex(faceDesc.Cullface)
			}

			model.Faces = append(model.Faces, bakedFace)
		}
	}

	model.FaceCullingMask = faceCullingMask
	model.RenderType = resolvedRenderType

	// fmt.Printf("[baker] model %s resolved to rendertype=%d\n", modelId, resolvedRenderType)

	bakedModelCache[modelId] = model

	return model, nil
}

func rotateFace(quadVerts [4]rl.Vector3, normal rl.Vector3, rotation block.ModelRotation) (outVerts [4]rl.Vector3, outNorm rl.Vector3) {
	var axis rl.Vector3
	switch rotation.Axis {
	case "x":
		{
			axis = rl.NewVector3(1, 0, 0)
		}
	case "y":
		{
			axis = rl.NewVector3(0, 1, 0)
		}
	case "z":
		{
			axis = rl.NewVector3(0, 0, 1)
		}
	}

	angleRad := rotation.Angle * rl.Deg2rad

	mTransformNegative := rl.MatrixTranslate(-rotation.Origin.X, -rotation.Origin.Y, -rotation.Origin.Z)
	mRotate := rl.MatrixRotate(axis, angleRad)
	mTransformPositive := rl.MatrixTranslate(rotation.Origin.X, rotation.Origin.Y, rotation.Origin.Z)

	quadM := rl.MatrixMultiply(mTransformNegative, mRotate)
	normM := mRotate

	if rotation.Rescale {
		scaleFactor := float32(1 / math.Cos(float64(angleRad)))
		var scaleVec rl.Vector3
		switch rotation.Axis {
		case "x":
			{
				scaleVec = rl.NewVector3(1, scaleFactor, scaleFactor)
			}
		case "y":
			{
				scaleVec = rl.NewVector3(scaleFactor, 1, scaleFactor)
			}
		case "z":
			{
				scaleVec = rl.NewVector3(scaleFactor, scaleFactor, 1)
			}
		}
		mRescale := rl.MatrixScale(scaleVec.X, scaleVec.Y, scaleVec.Z)
		quadM = rl.MatrixMultiply(quadM, mRescale)
		// Normal vector must be transformed by the inverse transpose of the
		// rescale matrix, since it is a non-uniform transformation.
		normM = rl.MatrixMultiply(normM, mRescale)
		normM = rl.MatrixInvert(normM)
		normM = rl.MatrixTranspose(normM)
	}
	quadM = rl.MatrixMultiply(quadM, mTransformPositive)
	for i := 0; i < 4; i++ {
		outVerts[i] = rl.Vector3Transform(quadVerts[i], quadM)
	}

	outNorm = rl.Vector3Transform(normal, normM)
	return
}

// Rearranges the coordinates which determine a cuboid bounding box into a
// minimum point and a maximum point
func sortCuboidPoints(from, to rl.Vector3) (rl.Vector3, rl.Vector3) {
	var minX, minY, minZ float32 = 999, 999, 999
	var maxX, maxY, maxZ float32 = -999, -999, -999

	if from.X < minX {
		minX = from.X
	}
	if from.Y < minY {
		minY = from.Y
	}
	if from.Z < minZ {
		minZ = from.Z
	}
	if from.X > maxX {
		maxX = from.X
	}
	if from.Y > maxY {
		maxY = from.Y
	}
	if from.Z > maxZ {
		maxZ = from.Z
	}

	if to.X < minX {
		minX = to.X
	}
	if to.Y < minY {
		minY = to.Y
	}
	if to.Z < minZ {
		minZ = to.Z
	}
	if to.X > maxX {
		maxX = to.X
	}
	if to.Y > maxY {
		maxY = to.Y
	}
	if to.Z > maxZ {
		maxZ = to.Z
	}

	return rl.Vector3{X: minX, Y: minY, Z: minZ}, rl.Vector3{X: maxX, Y: maxY, Z: maxZ}
}

// Returns the index of the first occurence of target in the list.
func findIndex(list [4]rl.Vector3, target rl.Vector3) int {
	const epsilon = 1e-4

	closeTo := func(a, b float32) bool {
		return math.Abs(float64(a)-float64(b)) <= epsilon
	}

	for i, candidate := range list {
		if closeTo(candidate.X, target.X) && closeTo(candidate.Y, target.Y) && closeTo(candidate.Z, target.Z) {
			return i
		}
	}
	return -1
}

// Compiles the loaded blockstate assets into mesher-compatible models.
func BakeBlockStates(atlas *TextureAtlas, assets *block.BlockAssets) BakedBlockStateMap {
	m := make(BakedBlockStateMap)
	bakedModelCache = make(map[string]BakedModel)

	for resourceId, blockStateDesc := range assets.BlockStates {
		blockState := BakedBlockState{}
		if len(blockStateDesc.Multipart) > 0 {
			// If the multipart field is present, this is a multipart block state
			blockState.IsMultipart = true
			// TODO: process multipart conditions
		} else {
			blockState.IsMultipart = false
			variantMap := make(map[string]BakedVariant)
			for variantKey, variantDesc := range blockStateDesc.Variants {
				variant := BakedVariant{}
				if len(variantDesc) > 1 {
					// always true if there is more than one model, even if all variants
					// are weight=1
					variant.IsWeightedChoice = true
				}
				variant.Models = make([]BakedVariantModel, 0)
				for _, modelDesc := range variantDesc {
					modelWrapper := BakedVariantModel{
						Weight:    1,
						ModelName: modelDesc.Model,
						RotationX: modelDesc.X,
						RotationY: modelDesc.Y,
						Uvlock:    modelDesc.Uvlock,
					}

					// NOTE: we assume that a weight of 0 means the value was unset and should be
					// the default value, since a variant with 0 weight is meaningless. May need
					// to correct this assumption based on testing
					if modelDesc.Weight != 0 {
						modelWrapper.Weight = modelDesc.Weight
					}

					bakedModel, err := bakeModel(atlas, assets.Models, modelDesc.Model)
					if err != nil {
						fmt.Printf("error baking model for blockstate %s (variant \"%s\"): %s\n", resourceId, variantKey, err)
						continue
					}
					modelWrapper.Model = &bakedModel
					variant.Models = append(variant.Models, modelWrapper)
					variant.TotalWeight += modelWrapper.Weight
				}

				variantMap[variantKey] = variant
			}
			blockState.Variants = variantMap
		}
		m[resourceId] = blockState
		// variantKey, variantState := mapPickFirst(blockState.Variants)
		// if len(blockState.Variants) > 1 {
		// 	// TODO: variants are not yet supported. For not we will just pick one at random
		// 	log.Printf("[WARN] BlockState with resource id %s has multiple variants; we will pick one at random (%s).\n", resourceId, variantKey)
		// }

		// if len(variantState) > 1 {
		// 	// TODO: when multiple models are specified, one should be chosen at random. for now, we will pick the first.
		// 	log.Printf("[WARN] BlockStateVariant %s-%s has multiple models; we will pick the first one.\n", resourceId, variantKey)
		// }

		// variantModelDesc := variantState[0]

		// // TODO: for now, we are ignoring all variant properties as well (x,y,uv,weight)

		// blockModel, err := bakeModel(atlas, assets.Models, variantModelDesc.Model)
		// if err != nil {
		// 	fmt.Printf("error baking model for blockstate %s: %s\n", resourceId, err)
		// }

		// m[variantModelDesc.Model] = blockModel
	}

	return m
}

func mapPickFirst[K comparable, V any](m map[K]V) (K, V) {
	var noKey K
	var noVal V
	for k, v := range m {
		return k, v
	}
	return noKey, noVal
}
