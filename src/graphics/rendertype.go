package graphics

import (
	"math"

	"github.com/faideww/mc-iso/src/block"
)

type RenderType int

const (
	RTOpaque RenderType = iota
	RTCutout
	RTTranslucent
	RTAir
)

func computeTextureRenderType(uv block.Rect, pixelData PixelData) RenderType {
	currentRenderType := RTOpaque

	x0, y0, x1, y1 := sortRectPoints(uv)

	// uvs are described in 0-16 space, but block textures may be a different size if they contain texture data for multiple faces in one file. So we need to map the uvs back into real pixel space.

	wRatio := float64(pixelData.width) / 16.0
	hRatio := float64(pixelData.height) / 16.0

	pixelX0 := int(math.Round(x0 * wRatio))
	pixelX1 := int(math.Round(x1 * wRatio))
	pixelY0 := int(math.Round(y0 * hRatio))
	pixelY1 := int(math.Round(y1 * hRatio))

	for y := pixelY0; y < pixelY1; y++ {
		for x := pixelX0; x < pixelX1; x++ {
			colorIndex := y*int(pixelData.width) + x
			c := pixelData.data[colorIndex]
			if c.A == 255 {
				// Pixel is fully opaque; continue
			} else if c.A == 0 {
				// Pixel is fully transparent; switch to CUTOUT and continue
				currentRenderType = RTCutout
			} else {
				// Pixel is partially transparent; switch to TRANSLUCENT and exit (we
				// don't need to check any others)
				return RTTranslucent
			}
		}
	}

	return currentRenderType
}

// Returns the minimum and maximum x and y values in a Rect
func sortRectPoints(r block.Rect) (float64, float64, float64, float64) {
	var minX, minY float32 = 999, 999
	var maxX, maxY float32 = -999, -999

	if r.X1 < minX {
		minX = r.X1
	}
	if r.Y1 < minY {
		minY = r.Y1
	}
	if r.X1 > maxX {
		maxX = r.X1
	}
	if r.Y1 > maxY {
		maxY = r.Y1
	}

	if r.X2 < minX {
		minX = r.X2
	}
	if r.Y2 < minY {
		minY = r.Y2
	}
	if r.X2 > maxX {
		maxX = r.X2
	}
	if r.Y2 > maxY {
		maxY = r.Y2
	}

	return float64(minX), float64(minY), float64(maxX), float64(maxY)
}
