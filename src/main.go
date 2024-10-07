package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	rl.InitWindow(800, 450, "raylib [core] example - basic window")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.DrawText("Congrats! you created your first window!", 190, 200, 20, rl.LightGray)
		rl.EndDrawing()
	}

}
