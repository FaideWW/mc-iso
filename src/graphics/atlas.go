package graphics

import (
	"archive/zip"
	"io"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type TextureAtlas struct {
	Atlas    rl.Texture2D
	UVMap    map[string]rl.Rectangle
	TileSize int
}

// NOTE: for now, we are only loading block textures. This will probably expand to other
func LoadTextureAtlas(fileMap map[string]*zip.File, texPaths map[string]string, tileSize int) (TextureAtlas, error) {
	count := len(texPaths)
	// Approximation of a square atlas
	tilesPerRow := int(math.Ceil(math.Sqrt(float64(count))))

	atlasWidth := tileSize * tilesPerRow
	atlasHeight := tileSize * tilesPerRow

	atlasImg := rl.GenImageColor(atlasWidth, atlasHeight, rl.Pink)
	defer rl.UnloadImage(atlasImg)

	uvs := make(map[string]rl.Rectangle)

	texIndex := 0
	for texId, path := range texPaths {
		if file, ok := fileMap[path]; ok {
			rc, err := file.Open()
			if err != nil {
				return TextureAtlas{}, err
			}
			defer rc.Close()

			imageData, err := io.ReadAll(rc)
			if err != nil {
				return TextureAtlas{}, err
			}

			texImg := rl.LoadImageFromMemory(".png", imageData, int32(len(imageData)))

			if texImg.Width != int32(tileSize) || texImg.Height != int32(tileSize) {
				// TODO: warn that loaded texture is not the expected size, and skip
			} else {
				x := (texIndex % tilesPerRow) * tileSize
				y := (texIndex / tilesPerRow) * tileSize
				destRect := rl.NewRectangle(float32(x), float32(y), float32(tileSize), float32(tileSize))

				rl.ImageDraw(atlasImg, texImg, rl.NewRectangle(0, 0, float32(tileSize), float32(tileSize)), destRect, rl.White)

				uvs[texId] = rl.NewRectangle(
					destRect.X/float32(atlasWidth),
					destRect.Y/float32(atlasHeight),
					destRect.Width/float32(atlasWidth),
					destRect.Height/float32(atlasHeight),
				)
			}

			rl.UnloadImage(texImg)
		} else {
			// TODO: warn that requested texture is not found
		}

		texIndex++
	}
	rl.ExportImage(*atlasImg, "test.png")
	texture := rl.LoadTextureFromImage(atlasImg)

	return TextureAtlas{texture, uvs, tileSize}, nil
}
