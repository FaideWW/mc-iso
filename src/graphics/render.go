package graphics

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type BlockFaceTexture struct {
	TexId  string
	Uv     rl.Rectangle
	Normal rl.Vector3
}

type Block struct {
	Pos   rl.Vector3
	Faces [6]BlockFaceTexture
}

func DrawBlocksImmediate(camera *rl.Camera, atlas *TextureAtlas, blocks []Block) {
	rl.SetTexture(atlas.Atlas.ID)

	for _, b := range blocks {
		RenderBlockImmediate(atlas, b)
	}
	rl.SetTexture(0)
}

func RenderBlockImmediate(atlas *TextureAtlas, block Block) {
	rl.Begin(rl.Quads)
	rl.Color4ub(rl.White.R, rl.White.G, rl.White.B, rl.White.A)
	for face := 0; face < 6; face++ {
		blockFace := block.Faces[face]
		uv, ok := atlas.UVMap[blockFace.TexId]
		if !ok {
			// skip drawing this face if there is no texture for it.
			// TODO: should we change this behavior to draw a missing-texture texture?
			continue
		}

		verts := [4]rl.Vector3{}

		// Faces are wound clockwise from the top left corner, looking towards them
		// from the outside of the block
		switch face {
		case 0:
			{ // +X (east)
				verts = [4]rl.Vector3{
					{X: block.Pos.X + 1, Y: block.Pos.Y + 1, Z: block.Pos.Z},
					{X: block.Pos.X + 1, Y: block.Pos.Y + 1, Z: block.Pos.Z + 1},
					{X: block.Pos.X + 1, Y: block.Pos.Y, Z: block.Pos.Z + 1},
					{X: block.Pos.X + 1, Y: block.Pos.Y, Z: block.Pos.Z},
				}
			}
		case 1:
			{ // -X (west)
				verts = [4]rl.Vector3{
					{X: block.Pos.X, Y: block.Pos.Y + 1, Z: block.Pos.Z + 1},
					{X: block.Pos.X, Y: block.Pos.Y + 1, Z: block.Pos.Z},
					{X: block.Pos.X, Y: block.Pos.Y, Z: block.Pos.Z},
					{X: block.Pos.X, Y: block.Pos.Y, Z: block.Pos.Z + 1},
				}
			}
		case 2:
			{ // +Y (up)
				verts = [4]rl.Vector3{
					{X: block.Pos.X, Y: block.Pos.Y + 1, Z: block.Pos.Z + 1},
					{X: block.Pos.X + 1, Y: block.Pos.Y + 1, Z: block.Pos.Z + 1},
					{X: block.Pos.X + 1, Y: block.Pos.Y + 1, Z: block.Pos.Z},
					{X: block.Pos.X, Y: block.Pos.Y + 1, Z: block.Pos.Z},
				}
			}
		case 3:
			{ // -Y (down)
				verts = [4]rl.Vector3{
					{X: block.Pos.X, Y: block.Pos.Y, Z: block.Pos.Z},
					{X: block.Pos.X + 1, Y: block.Pos.Y, Z: block.Pos.Z},
					{X: block.Pos.X + 1, Y: block.Pos.Y, Z: block.Pos.Z + 1},
					{X: block.Pos.X, Y: block.Pos.Y, Z: block.Pos.Z + 1},
				}
			}
		case 4:
			{ // +Z (south)
				verts = [4]rl.Vector3{
					{X: block.Pos.X, Y: block.Pos.Y, Z: block.Pos.Z + 1},
					{X: block.Pos.X + 1, Y: block.Pos.Y, Z: block.Pos.Z + 1},
					{X: block.Pos.X + 1, Y: block.Pos.Y + 1, Z: block.Pos.Z + 1},
					{X: block.Pos.X, Y: block.Pos.Y + 1, Z: block.Pos.Z + 1},
				}
			}
		case 5:
			{ // -Z (north)
				verts = [4]rl.Vector3{
					{X: block.Pos.X + 1, Y: block.Pos.Y, Z: block.Pos.Z},
					{X: block.Pos.X, Y: block.Pos.Y, Z: block.Pos.Z},
					{X: block.Pos.X, Y: block.Pos.Y + 1, Z: block.Pos.Z},
					{X: block.Pos.X + 1, Y: block.Pos.Y + 1, Z: block.Pos.Z},
				}
			}
		}

		rl.TexCoord2f(uv.X, uv.Y)
		rl.Vertex3f(verts[0].X, verts[0].Y, verts[0].Z)

		rl.TexCoord2f(uv.X+uv.Width, uv.Y)
		rl.Vertex3f(verts[1].X, verts[1].Y, verts[1].Z)

		rl.TexCoord2f(uv.X+uv.Width, uv.Y+uv.Height)
		rl.Vertex3f(verts[2].X, verts[2].Y, verts[2].Z)

		rl.TexCoord2f(uv.X, uv.Y+uv.Height)
		rl.Vertex3f(verts[3].X, verts[3].Y, verts[3].Z)

	}
	rl.End()
}

func RenderCube(x, y, z float32, color rl.Color) {
	// rl.drawTexture
	rl.DrawCube(rl.Vector3{X: x, Y: y, Z: z},
		1.0, 1.0, 1.0,
		color)
}
