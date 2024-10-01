package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

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

	ParseNBT(f)

}
