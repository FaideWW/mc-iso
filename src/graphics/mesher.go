package graphics

import (
	"fmt"
	"unsafe"

	"github.com/faideww/mc-iso/src/block"
	"github.com/faideww/mc-iso/src/region"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Vertex struct {
	Pos    [3]float32
	Uv     [2]float32
	Normal [3]int8
}

type FaceDef struct {
	Normal  rl.Vector3
	Corners [4][3]int8 // offsets in model space from 0-1
}

func getNeighborCullingData(ctx *RenderContext, section *region.Section, nx, ny, nz int) (uint8, RenderType) {
	if model, ok := ctx.GetBakedModelAt(section, nx, ny, nz); ok {
		return model.FaceCullingMask, model.RenderType
	}

	return 0, RTAir
}

func getNeighborCoords(x, y, z int, face FaceDir) (int, int, int) {
	switch face {
	case FaceWest:
		{
			return x - 1, y, z
		}
	case FaceEast:
		{
			return x + 1, y, z
		}
	case FaceNorth:
		{
			return x, y, z - 1
		}
	case FaceSouth:
		{
			return x, y, z + 1
		}
	case FaceDown:
		{
			return x, y - 1, z
		}
	case FaceUp:
		{
			return x, y + 1, z
		}
	}

	return x, y, z
}

func BuildSectionMeshFromBakedModels(ctx *RenderContext, section *region.Section) *rl.Model {
	const maxFaces = 16 * 16 * 16 * 6
	const maxVerts = maxFaces * 4
	const maxIndices = maxFaces * 6

	// Allocate data slices
	verts := make([]float32, 0, maxVerts*3)
	texcoords := make([]float32, 0, maxVerts*2)
	normals := make([]float32, 0, maxVerts*3)
	indices := make([]uint16, 0, maxIndices)

	pushQuad := func(
		x, y, z float32, // world-space transform
		f BakedFace, // face verts and uv
	) {
		baseIndex := uint16(len(verts) / 3)

		// append vertex, uv, and normal data
		for i := 0; i < 4; i++ {
			cx := x + f.Vertices[i].X
			cy := y + f.Vertices[i].Y
			cz := z + f.Vertices[i].Z
			verts = append(verts, cx, cy, cz)
			texcoords = append(texcoords, f.Uv[i].X, f.Uv[i].Y)
			normals = append(normals, f.Normal.X, f.Normal.Y, f.Normal.Z)
		}

		// append two tris to the indices
		indices = append(indices,
			baseIndex, baseIndex+1, baseIndex+2,
			baseIndex, baseIndex+2, baseIndex+3)
	}

	// Sweep each block in the region
	for y := 0; y < 16; y++ {
		for z := 0; z < 16; z++ {
			for x := 0; x < 16; x++ {
				model, ok := ctx.GetBakedModelAt(section, x, y, z)
				// ok will be false if the block is air, or if no model was found
				if !ok {
					fmt.Printf("block at %d,%d,%d is air or missing\n", x, y, z)
					continue
				}

				renderType := model.RenderType

				for _, face := range model.Faces {
					// pushQuad(float32(x), float32(y), float32(z), face)
					// continue

					if face.Cullface == -1 {
						// if cullface is -1, we always render the face
						pushQuad(float32(x), float32(y), float32(z), face)
						continue
					}

					nx, ny, nz := getNeighborCoords(x, y, z, face.Cullface)

					neighborMask, neighborRenderType := getNeighborCullingData(ctx, section, nx, ny, nz)
					neighborFace := getOpposingFace(face.Cullface)

					neighborCoversFace := (neighborMask & (1 << neighborFace)) != 0

					// If the neighboring face doesn't completely cover the current face, we draw it
					if !neighborCoversFace {
						pushQuad(float32(x), float32(y), float32(z), face)
						continue
					}

					// Otherwise, we test the neighbor's render type
					shouldCull := false
					if neighborRenderType == RTOpaque {
						shouldCull = true
					} else if neighborRenderType == renderType {
						// Matching render types cull each other (TODO: confirm this is true)
						shouldCull = true
					}

					if !shouldCull {
						pushQuad(float32(x), float32(y), float32(z), face)
					}
				}
			}
		}
	}
	// Marshal into rl.Mesh
	mesh := rl.Mesh{}
	mesh.VertexCount = int32(len(verts) / 3)
	mesh.TriangleCount = int32(len(indices) / 3)

	fmt.Printf("[Mesher] mesh stats: Vertices=%d, Triangles=%d\n", mesh.VertexCount, mesh.TriangleCount)

	mesh.Vertices = unsafe.SliceData(verts)
	mesh.Texcoords = unsafe.SliceData(texcoords)
	mesh.Normals = unsafe.SliceData(normals)
	mesh.Indices = unsafe.SliceData(indices)

	rl.UploadMesh(&mesh, false) // Static draw

	model := rl.LoadModelFromMesh(mesh)
	model.GetMaterials()[0].GetMap(rl.MapDiffuse).Texture = ctx.TextureAtlas.Atlas

	return &model
}

func ClearMesh(model rl.Model) {
	// Vertices, Normals and Texcoords of your CUSTOM mesh are Go slices.
	// UnloadModel calls UnloadMesh for every mesh and UnloadMesh tries
	// to free your Go slices. This will panic because it cannot free
	// Go slices. Free() is a C function and it expects to free C memory
	// and not a Go slice. So clear the slices manually like this.
	model.Meshes.Vertices = nil
	model.Meshes.Normals = nil
	model.Meshes.Texcoords = nil
}

// --- Older code begins here ---

// Hard-coded cube faces
var faces = [...]FaceDef{
	/* -X (west) */ {rl.NewVector3(-1, 0, 0), [4][3]int8{{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}}},
	/* +X (east) */ {rl.NewVector3(1, 0, 0), [4][3]int8{{1, 0, 1}, {1, 0, 0}, {1, 1, 0}, {1, 1, 1}}},
	/* -Z (north)*/ {rl.NewVector3(0, 0, -1), [4][3]int8{{1, 0, 0}, {0, 0, 0}, {0, 1, 0}, {1, 1, 0}}},
	/* +Z (south)*/ {rl.NewVector3(0, 0, 1), [4][3]int8{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}}},
	/* -Y (down) */ {rl.NewVector3(0, -1, 0), [4][3]int8{{0, 0, 1}, {0, 0, 0}, {1, 0, 0}, {1, 0, 1}}},
	/* +Y (up)   */ {rl.NewVector3(0, 1, 0), [4][3]int8{{0, 1, 0}, {0, 1, 1}, {1, 1, 1}, {1, 1, 0}}},
}

// Returns a bitmask of whether blocks neighboring the given position are
// occupied, or vacant (for the purposes of generating face geometry). A 1 in
// the bit position denotes that the neighboring block is opaque, and a face
// should not be drawn.
// The bits are ordered as follows:
// Face 0  0 +Y -Y +Z -Z +X -X
// Bit  7  6  5  4  3  2  1  0
func neighborMask(s *region.Section, x, y, z int) uint8 {
	// TODO: this only checks neighbors within the section; for a complete map rendering, we'll have to check neighboring sections/chunks as well.
	var m uint8
	if s.IsOpaque(x-1, y, z) {
		m |= 1 << 0
	}
	if s.IsOpaque(x+1, y, z) {
		m |= 1 << 1
	}
	if s.IsOpaque(x, y, z-1) {
		m |= 1 << 2
	}
	if s.IsOpaque(x, y, z+1) {
		m |= 1 << 3
	}
	if s.IsOpaque(x, y-1, z) {
		m |= 1 << 4
	}
	if s.IsOpaque(x, y+1, z) {
		m |= 1 << 5
	}
	return m
}

func BuildSectionMesh(atlas *TextureAtlas, section *region.Section) *rl.Model {
	// sections are 16x16x16 sub-chunks
	const maxFaces = 16 * 16 * 16 * 6
	const maxVerts = maxFaces * 4
	const maxIndices = maxFaces * 6

	// Allocate data slices
	verts := make([]float32, 0, maxVerts*3)
	texcoords := make([]float32, 0, maxVerts*2)
	normals := make([]float32, 0, maxVerts*3)
	indices := make([]uint16, 0, maxIndices)

	// Appends block face data to the slices
	pushQuad := func(
		x, y, z int, // world-space transform
		f *FaceDef,
		uv rl.Rectangle, // atlas texcoords (already in 0-1 uv format)
	) {

		// Tex coords
		u0 := uv.X
		v0 := uv.Y
		u1 := uv.X + uv.Width
		v1 := uv.Y + uv.Height

		uvs := [4][2]float32{{u0, v1}, {u1, v1}, {u1, v0}, {u0, v0}}

		baseIndex := uint16(len(verts) / 3)

		// append vertex, uv, and normal data
		for i := 0; i < 4; i++ {
			cx := float32(x + int(f.Corners[i][0]))
			cy := float32(y + int(f.Corners[i][1]))
			cz := float32(z + int(f.Corners[i][2]))
			verts = append(verts, cx, cy, cz)
			texcoords = append(texcoords, uvs[i][0], uvs[i][1])
			normals = append(normals, f.Normal.X, f.Normal.Y, f.Normal.Z)
		}

		// append two tris to the indices
		indices = append(indices,
			baseIndex, baseIndex+1, baseIndex+2,
			baseIndex, baseIndex+2, baseIndex+3)
	}

	// Sweep each block in the region
	for y := 0; y < 16; y++ {
		for z := 0; z < 16; z++ {
			for x := 0; x < 16; x++ {
				// Skip air blocks
				paletteIndex, err := section.GetPaletteIdx(x, y, z)
				if err != nil {
					fmt.Println(err)
					continue
				}
				// fmt.Printf("block at %d,%d,%d: %+v", x, y, z, paletteIndex)
				if paletteIndex == section.BlockStates.AirIdx {
					// fmt.Printf("(air)\n")
					continue
				}
				blockResourceName := section.BlockStates.Palette[paletteIndex].Name
				namespace, blockName := block.DecodeResourceId(blockResourceName)

				blockTextureName := fmt.Sprintf("%s:block/%s", namespace, blockName)
				// fmt.Printf("(%s) ", blockTextureName)

				// TODO: accommodate block models with different textures per face
				// TODO: accommodate sparse block models
				uvRect := atlas.UVMap[blockTextureName]
				// fmt.Printf("(uv: %v) ", uvRect)

				// For each cardinal direction, emit a face quad if the neighboring block is empty (either at a section boundary, or if the neighbor is air
				neighbors := neighborMask(section, x, y, z)
				// fmt.Printf("(nb: %08b) ", neighbors)

				// For each face, see if new geometry is needed
				for f := 0; f < 6; f++ {
					if neighbors&(1<<f) == 0 {
						// If the neighbor is not opaque, add a quad
						pushQuad(x, y, z, &faces[f], uvRect)
					}
				}
				// fmt.Println()
			}
		}
	}

	// Marshal into rl.Mesh
	mesh := rl.Mesh{}
	mesh.VertexCount = int32(len(verts) / 3)
	mesh.TriangleCount = int32(len(indices) / 3)

	mesh.Vertices = unsafe.SliceData(verts)
	mesh.Texcoords = unsafe.SliceData(texcoords)
	mesh.Normals = unsafe.SliceData(normals)
	mesh.Indices = unsafe.SliceData(indices)

	rl.UploadMesh(&mesh, false) // Static draw

	model := rl.LoadModelFromMesh(mesh)
	model.GetMaterials()[0].GetMap(rl.MapDiffuse).Texture = atlas.Atlas

	return &model
}
