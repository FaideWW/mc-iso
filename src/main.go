package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	args := os.Args

	if len(args) < 2 {
		log.Fatal("missing path to world dir")
	}

	inDir := args[1]

	fmt.Printf("inDir: %s\n", inDir)
}
