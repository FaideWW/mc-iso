package main

import (
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/faideww/mc-iso/src/nbt"
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

	f, err := os.Open(levelDatPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	isCompressed, err := nbt.CheckCompression(f)
	if err != nil {
		log.Fatal(err)
	}

	var decompressed io.Reader

	switch isCompressed {
	case nbt.Uncompressed:
		fmt.Printf("file is uncompressed\n")
		decompressed = f
	case nbt.Gzip:
		fmt.Printf("using gzip compression\n")
		decompressed, err = gzip.NewReader(f)
		if err != nil {
			log.Fatal(err)
		}
	case nbt.Zlib:
		fmt.Printf("using zlib compression\n")
		decompressed, err = zlib.NewReader(f)
		if err != nil {
			log.Fatal(err)
		}
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
