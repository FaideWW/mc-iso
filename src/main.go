package main

/**

TODO: textures extraction from an on-disk client.jar (default install locations and user-specified paths)
TODO: performance comparison between loading individual texture images vs. using an atlas


*/

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/faideww/mc-iso/src/block"
	"github.com/faideww/mc-iso/src/nbt"
	"github.com/faideww/mc-iso/src/region"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Level struct {
	Data LevelData `nbt:"Data"`
}

type VersionData struct {
	Id   int    `nbt:"Id"`
	Name string `nbt:"Name"`
}

type LevelData struct {
	AllowCommands        bool    `nbt:"allowCommands"`
	BorderCenterX        float64 `nbt:"BorderCenterX"`
	BorderCenterY        float64 `nbt:"BorderCenterY"`
	BorderDamgePerBlock  float64 `nbt:"BorderDamgePerBlock"`
	BorderSize           float64 `nbt:"BorderSize"`
	BorderSafeZone       float64 `nbt:"BorderSafeZone"`
	BorderSizeLerpTarget float64 `nbt:"BorderSizeLerpTarget"`
	BorderSizeLerpTime   int64   `nbt:"BorderSizeLerpTime"`
	BorderWarningBlocks  float64 `nbt:"BorderWarningBlocks"`
	BorderWarningTime    float64 `nbt:"BorderWarningTime"`
	ClearWeatherTime     int     `nbt:"ClearWeatherTime"`

	LevelName string      `nbt:"LevelName"`
	SpawnX    int         `nbt:"SpawnX"`
	SpawnY    int         `nbt:"SpawnY"`
	SpawnZ    int         `nbt:"SpawnZ"`
	Version   VersionData `nbt:"Version"`
	WasModded bool        `nbt:"WasModded"`
}

func main() {
	args := os.Args

	if len(args) < 2 {
		log.Fatal("missing path to world dir")
	}

	worldPath := args[1]

	fmt.Printf("worldPath: %s\n", worldPath)

	levelDatPath := filepath.Join(worldPath, "level.dat")

	levelFile, err := os.Open(levelDatPath)
	if err != nil {
		panic(err)
	}
	defer levelFile.Close()

	decompressed, err := nbt.Decompress(levelFile)
	if err != nil {
		log.Fatal(err)
	}

	decoder := nbt.NewDecoder(decompressed)

	var result Level
	name, err := decoder.Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("parsed level.dat - top level tagname: %q\n", name)
	fmt.Printf("%+v\n", result)

	regionFilePath := filepath.Join(worldPath, "region/r.0.0.mca")
	regionFile, err := os.Open(regionFilePath)
	if err != nil {
		panic(err)
	}
	defer regionFile.Close()

	reg, err := region.NewRegion(regionFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("successfully parsed region\n")
	// fmt.Printf("example chunk 0: %+v\n", region.Chunks[0])

	for i, s := range reg.Chunks[0].Sections {
		fmt.Printf("section %d Y: %d\n", i, s.Y)
	}

	debugPrintChunkSection(reg.Chunks[0].Sections[0])

	tmpdir, err := os.MkdirTemp("", "mc-iso-assets")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	screenWidth := int32(800)
	screenHeight := int32(450)

	rl.InitWindow(screenWidth, screenHeight, "raylib [core] example - basic window")
	defer rl.CloseWindow()

	assets, err := block.UnpackAssetsFromJar("/Users/faide/Library/Application Support/minecraft/versions/1.21.4/1.21.4.jar", []string{"minecraft:bedrock"}, tmpdir)
	if err != nil {
		log.Fatal(err)
	}

	rl.SetTargetFPS(60)

	camera := rl.Camera{
		Position:   rl.Vector3{X: -10, Y: 10, Z: -10},
		Target:     rl.Vector3{X: 0, Y: 0, Z: 0},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       10.0,
		Projection: rl.CameraOrthographic,
	}
	orientationHintTex := rl.LoadRenderTexture(50, 50)
	rl.BeginTextureMode(orientationHintTex)
	rl.ClearBackground(rl.Black)
	rl.EndTextureMode()

	var prevMouseRay rl.Ray

	for !rl.WindowShouldClose() {
		// Update mouse
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			prevMouseRay = rl.GetMouseRay(rl.GetMousePosition(), camera)
		} else if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			currMouseRay := rl.GetMouseRay(rl.GetMousePosition(), camera)
			rayDiff := rl.Vector3Subtract(currMouseRay.Position, prevMouseRay.Position)
			if rayDiff.X != 0 || rayDiff.Y != 0 || rayDiff.Z != 0 {
				fmt.Printf("mouse moved world (X:%0.2f, Y:%0.2f, Z:%0.2f)\n", rayDiff.X, rayDiff.Y, rayDiff.Z)

				camera.Position = rl.Vector3Subtract(camera.Position, rayDiff)
				camera.Target = rl.Vector3Subtract(camera.Target, rayDiff)

			}
			prevMouseRay = currMouseRay
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)
		rl.BeginMode3D(camera)
		rl.DrawGrid(10, 1)
		rl.DrawTexture(assets.Textures["minecraft:block/bedrock"], 0, 0, rl.White)
		// render.RenderCube(-100, 0, 0, rl.Yellow)
		// render.RenderCube(1, 0, 0, rl.Red)
		// render.RenderCube(1, 0, 1, rl.Green)
		// render.RenderCube(0, 0, 1, rl.Blue)
		rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: 0}, rl.Vector3{X: 1, Y: 0, Z: 0}, rl.Red)
		rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: 0}, rl.Vector3{X: 0, Y: 1, Z: 0}, rl.Green)
		rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: 0}, rl.Vector3{X: 0, Y: 0, Z: 1}, rl.Blue)

		rl.EndMode3D()

		cameraRay := rl.GetCameraForward(&camera)
		vecOrigin := rl.Vector3{}

		rl.BeginTextureMode(orientationHintTex)

		axisCamera := rl.Camera{
			Position:   rl.Vector3Subtract(vecOrigin, cameraRay),
			Target:     vecOrigin,
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       2.0,
			Projection: rl.CameraOrthographic,
		}

		rl.BeginMode3D(axisCamera)
		rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: 0}, rl.Vector3{X: 1, Y: 0, Z: 0}, rl.Red)
		rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: 0}, rl.Vector3{X: 0, Y: 1, Z: 0}, rl.Green)
		rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: 0}, rl.Vector3{X: 0, Y: 0, Z: 1}, rl.Blue)
		rl.EndMode3D()
		rl.EndTextureMode()

		rl.DrawTextureRec(orientationHintTex.Texture, rl.NewRectangle(0, 0, 50, -50),
			rl.NewVector2(float32(screenWidth)-50, 0), rl.White)

		rl.DrawFPS(10, 10)
		rl.EndDrawing()
	}
}

func debugPrintChunkSection(s region.Section) {
	fmt.Printf("section Y: %d\n", s.Y)
	fmt.Printf("biome palette (size:%d): %+v\n", len(s.Biomes.Palette), s.Biomes.Palette)
	fmt.Printf("block palette (size:%d): %+v\n", len(s.BlockStates.Palette), s.BlockStates.Palette)
	fmt.Printf("block data size:%d\n", len(s.BlockStates.Data))
	if region.IntPow(2, 4) > len(s.BlockStates.Palette) {
		fmt.Printf("index size: 4bit - %d bytes\n", (4*4096)/8)
	}

	fmt.Printf("palette indices: [ ")
	for i := 0; i < 4096; i++ {
		idx, err := s.BlockStates.Index(i, true)
		if err != nil {
			log.Fatal(err)
		}
		block := s.BlockStates.Palette[idx].Name
		fmt.Printf("%s ", block)
	}
	fmt.Printf("]\n")

	// fmt.Printf("palette data (size:%d elems, %d bytes): %+v\n", len(s.BlockStates.Data), len(s.BlockStates.Data)*8, s.BlockStates.Data)
}
