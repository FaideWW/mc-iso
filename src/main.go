package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/faideww/mc-iso/src/nbt"
)

type Level struct {
	Data LevelData `nbt:"Data"`
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
}

func main() {
	args := os.Args

	if len(args) < 2 {
		log.Fatal("missing path to world dir")
	}

	worldPath := args[1]

	fmt.Printf("worldPath: %s\n", worldPath)

	levelDatPath := filepath.Join(worldPath, "level.dat")

	f, err := os.Open(levelDatPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decompressed, err := gzip.NewReader(f)
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

}
