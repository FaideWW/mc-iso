package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/faideww/mc-iso/src/block"
	"github.com/faideww/mc-iso/src/graphics"
	"github.com/faideww/mc-iso/src/nbt"
	"github.com/faideww/mc-iso/src/region"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/joho/godotenv"
)

const DEBUG_DRAW_AXIS = true

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

func loadEnv() {
	env := os.Getenv("VISTAS_ENV")
	if "" == env {
		env = "development"
	}

	godotenv.Load(".env." + env + ".local")
	if "test" != env {
		godotenv.Load(".env.local")
	}

	godotenv.Load(".env." + env)
	godotenv.Load() // plain old .env
}

func main() {
	loadEnv()

	jarPath := os.Getenv("MC_JAR_PATH")
	worldPath := os.Getenv("WORLD_PATH")

	if "" == jarPath {
		panic(errors.New("No jar path provided (MC_JAR_PATH)"))
	}

	// Prefer to load world path from command line arguments, if it's provided
	if len(os.Args) > 1 {
		worldPath = os.Args[1]
	}

	if "" == worldPath {
		panic(errors.New("No world  provided (WORLD_PATH)"))
	}

	fmt.Printf("worldPath: %s\n", worldPath)
	fmt.Printf("jarPath: %s\n", jarPath)

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

	// TODO: for testing and debugging purposes, we isolate a single chunk to decode and render

	chunk := reg.Chunks[1]
	for i, s := range chunk.Sections {
		fmt.Printf("section %d Y: %d\n", i, s.Y)
	}

	blocksToUnpack := []string{}
	for _, s := range chunk.Sections {
		for _, block := range s.BlockStates.Palette {
			blocksToUnpack = append(blocksToUnpack, block.Name)
		}
	}

	log.Printf("blocks to unpack: %+v\n", blocksToUnpack)
	log.Printf("unpacking assets from jar...\n")

	jarReader, err := zip.OpenReader(jarPath)
	if err != nil {
		log.Printf("failed to open jar reader.\n")
		log.Fatal(err)
	}

	defer jarReader.Close()

	assets, err := block.UnpackAssetsFromJar(&jarReader.Reader, blocksToUnpack)

	if err != nil {
		log.Printf("failed to unpack assets.\n")
		log.Fatal(err)
	}
	log.Printf("unpacking complete.\n")

	screenWidth := int32(1920)
	screenHeight := int32(1080)
	// InitWindow *must* be called before textures are loaded
	rl.InitWindow(screenWidth, screenHeight, "raylib [core] example - basic window")
	defer rl.CloseWindow()

	log.Printf("building texture atlas...\n")
	textureAtlas, err := graphics.LoadTextureAtlas(assets.FileMap, assets.Textures, 16)
	if err != nil {
		log.Printf("failed to build atlas.\n")
		log.Fatal(err)
	}

	log.Printf("atlas built.\n")
	fmt.Printf("atlas uv map: %+v\n", textureAtlas.UVMap)

	log.Printf("baking models... ")
	// TODO: capture this return
	bakedModels := graphics.BakeBlockModels(&textureAtlas, assets)
	log.Printf("done.\n")

	log.Printf("initing renderer...\n")

	rl.SetTargetFPS(144)

	camera := rl.Camera{
		Position:   rl.Vector3{X: 100, Y: 100, Z: 100},
		Target:     rl.Vector3{X: 0, Y: 0, Z: 0},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       16.0,
		Projection: rl.CameraOrthographic,
	}

	renderCtx := graphics.RenderContext{
		Assets:       assets,
		BakedModels:  bakedModels,
		TextureAtlas: &textureAtlas,
	}

	models := make([]*rl.Model, len(chunk.Sections))

	model := graphics.BuildSectionMeshFromBakedModels(&renderCtx, &chunk.Sections[0])
	models[0] = model
	defer rl.UnloadModel(*model)
	defer graphics.ClearMesh(*model)

	// for i, section := range chunk.Sections {
	// 	model := graphics.BuildSectionMeshFromBakedModels(&renderCtx, &section)
	// 	models[i] = model
	// 	defer rl.UnloadModel(*model)
	// 	defer graphics.ClearMesh(*model)
	// }

	debugPrintChunkSection(chunk.Sections[0])
	dumpBakedModels(bakedModels)

	rl.DisableCursor()
	for !rl.WindowShouldClose() {
		rl.UpdateCamera(&camera, rl.CameraThirdPerson)

		rl.BeginDrawing()

		rl.ClearBackground(rl.Black)
		rl.BeginMode3D(camera)

		// rl.DrawGrid(16, 1)

		sectionY := 0 // chunk.Sections[0].Y * 16
		rl.DrawModelEx(*model, rl.NewVector3(0, float32(sectionY), 0), rl.NewVector3(0, 1, 0), 0, rl.NewVector3(1, 1, 1), rl.White)

		// for i, model := range models {
		// 	sectionY := chunk.Sections[i].Y * 16
		// 	rl.DrawModelEx(*model, rl.NewVector3(0, float32(sectionY), 0), rl.NewVector3(0, 1, 0), 0, rl.NewVector3(1, 1, 1), rl.White)
		// }

		if DEBUG_DRAW_AXIS {
			// Draw axis gizmo with depth testing disabled so that it always appears on top
			rl.DrawRenderBatchActive()
			rl.DisableDepthTest()
			drawAxisGizmo()
			rl.DrawRenderBatchActive()
			rl.EnableDepthTest()
		}
		rl.EndMode3D()

		rl.DrawFPS(10, 10)
		rl.EndDrawing()
	}
}

func drawAxisGizmo() {
	const L = 2.0 // axis length in world units
	rl.PushMatrix()
	rl.DrawLine3D(rl.Vector3Zero(), rl.NewVector3(L, 0, 0), rl.Red)
	rl.DrawLine3D(rl.Vector3Zero(), rl.NewVector3(0, L, 0), rl.Green)
	rl.DrawLine3D(rl.Vector3Zero(), rl.NewVector3(0, 0, L), rl.Blue)
	rl.PopMatrix()
}

func dumpBakedModels(modelMap graphics.ModelMap) {
	f, err := os.Create("model_dump.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	for resourceId, model := range modelMap {
		str := fmt.Sprintf("%s: %+v\n", resourceId, model)
		w.WriteString(str)
	}
	w.Flush()
}

func debugPrintChunkSection(s region.Section) {
	fmt.Printf("section Y: %d\n", s.Y)
	fmt.Printf("biome palette (size:%d): %+v\n", len(s.Biomes.Palette), s.Biomes.Palette)
	fmt.Printf("block palette (size:%d): %+v\n", len(s.BlockStates.Palette), s.BlockStates.Palette)
	fmt.Printf("block data size:%d\n", len(s.BlockStates.Data))
	fmt.Printf("index size: %dbit\n", s.BlockStates.IndexSize)
	fmt.Printf("air index: %d\n", s.BlockStates.AirIdx)
	// if region.IntPow(2, 4) > len(s.BlockStates.Palette) {
	// 	fmt.Printf("index size: 4bit - %d bytes\n", (4*4096)/8)
	// }

	// fmt.Printf("palette indices: [ ")
	// for i := 0; i < 4096; i++ {
	// 	idx, err := s.BlockStates.Index(i, true)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	block := s.BlockStates.Palette[idx].Name
	// 	fmt.Printf("%s (%d) ", block, idx)
	// }
	// fmt.Printf("]\n")

	// fmt.Printf("palette data (size:%d elems, %d bytes): %+v\n", len(s.BlockStates.Data), len(s.BlockStates.Data)*8, s.BlockStates.Data)
}
