package render

import rl "github.com/gen2brain/raylib-go/raylib"

func DumpModelData(resourcePath string) {

}

func RenderCube(x, y, z float32, color rl.Color) {
	rl.DrawCube(rl.Vector3{X: x, Y: y, Z: z},
		1.0, 1.0, 1.0,
		color)
}
