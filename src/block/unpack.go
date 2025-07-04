package block

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type BlockAssets struct {
	BlockStates map[string]BlockState
	Models      map[string]BlockModel
	Textures    map[string]rl.Texture2D
}

// Extracts block states, block models, and textures from a minecraft client.jar. Returns an assets struct containing the extracted data.
//
// For now, we only unpack resources for the `minecraft` namespace, but we
// should expand this to other resource packs in the future.
// NOTE: this function calls `rl.LoadTexture`, which requires an OpenGL context to be initialized. Do not call this until after `rl.InitWindow` has finished!
func UnpackAssetsFromJar(jarPath string, blocksToUnpack []string, destPath string) (*BlockAssets, error) {
	assets := BlockAssets{
		make(map[string]BlockState),
		make(map[string]BlockModel),
		make(map[string]rl.Texture2D),
	}

	targetMap := make(map[string]struct{})

	for _, resourceName := range blocksToUnpack {
		targetMap[resourceName] = struct{}{}
	}

	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil, err
	}

	defer r.Close()

	fmt.Printf("files in archive: %d\n", len(r.File))

	fileMap := make(map[string]*zip.File)

	for _, file := range r.File {
		fileMap[file.Name] = file
	}

	pendingModels := make(map[string]struct{})

	// Blockstates are the entrypoint for block data. We first need to unpack the
	// blockstate data in order to identify which models to retrieve.

	for len(blocksToUnpack) > 0 {
		var nextBlock string
		nextBlock, blocksToUnpack = blocksToUnpack[0], blocksToUnpack[1:]

		// If we've already unpacked this block, move on
		if _, ok := assets.BlockStates[nextBlock]; ok {
			continue
		}

		// If there is no namespace on the blockname, we assume the default of "minecraft"
		resourceName := strings.Split(nextBlock, ":")
		var namespace, blockName string
		if len(resourceName) == 1 {
			namespace = "minecraft"
			blockName = resourceName[0]
		} else {
			namespace = resourceName[0]
			blockName = resourceName[1]
		}

		filePath := fmt.Sprintf("assets/%s/blockstates/%s.json", namespace, blockName)

		if file, ok := fileMap[filePath]; ok {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}

			fmt.Printf("file: %s\n", file.Name)
			var block BlockState
			err = json.NewDecoder(rc).Decode(&block)
			if err != nil {
				fmt.Printf("could not decode json: %s\n", err)
				return nil, err
			}

			fmt.Printf("blockstate:\n%+v\n", block)

			assets.BlockStates[nextBlock] = block

			// Enqueue any new models to be unpacked
			if len(block.Variants) > 0 {
				for _, variant := range block.Variants {
					for _, model := range variant {
						if _, ok := pendingModels[model.Model]; !ok {
							pendingModels[model.Model] = struct{}{}
						}
					}
				}
			} else if len(block.Multipart) > 0 {
				for _, mpCase := range block.Multipart {
					for _, model := range mpCase.Apply {
						if _, ok := pendingModels[model.Model]; !ok {
							pendingModels[model.Model] = struct{}{}
						}
					}
				}

			}

			// if block.Parent != "" {
			// 	blocksToUnpack = append(blocksToUnpack, block.Parent)
			// }
		} else {
			fmt.Printf("no resource \"%s\" found at path %s\n", blockName, filePath)
		}
	}

	fmt.Printf("models found: %+v\n", pendingModels)

	pendingTextures := make(map[string]struct{})

	// Build the pending models queue, and then unpack them
	modelQueue := []string{}
	for modelName := range pendingModels {
		modelQueue = append(modelQueue, modelName)
	}

	for len(modelQueue) > 0 {
		nextModel := modelQueue[0]
		modelQueue = modelQueue[1:]

		// If we've already unpacked this block, move on
		if _, ok := assets.Models[nextModel]; ok {
			continue
		}

		// If there is no namespace on the model name, we assume the default of "minecraft"
		resourceName := strings.Split(nextModel, ":")
		var namespace, modelName string
		if len(resourceName) == 1 {
			namespace = "minecraft"
			modelName = resourceName[0]
		} else {
			namespace = resourceName[0]
			modelName = resourceName[1]
		}

		filePath := fmt.Sprintf("assets/%s/models/%s.json", namespace, modelName)

		if file, ok := fileMap[filePath]; ok {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			fmt.Printf("file: %s\n", file.Name)
			var model BlockModel
			err = json.NewDecoder(rc).Decode(&model)
			if err != nil {
				fmt.Printf("could not decode json: %s\n", err)
				return nil, err
			}

			fmt.Printf("%s:\n%+v\n", nextModel, model)

			// Enqueue any textures we find
			for _, texName := range model.Textures {
				if !strings.HasPrefix(texName, "#") {
					if _, ok := pendingTextures[texName]; !ok {
						pendingTextures[texName] = struct{}{}
					}
				}
			}

			assets.Models[nextModel] = model

			// Models can depend on other models, so we recursively unpack those as well
			if model.Parent != "" {
				modelQueue = append(modelQueue, model.Parent)
			}
		} else {
			fmt.Printf("no resource \"%s\" found at path %s\n", modelName, filePath)
		}
	}

	fmt.Printf("unpacked models: \n%+v\n", assets.Models)

	// Build the texture queue and unpack
	textureQueue := []string{}
	for tex := range pendingTextures {
		textureQueue = append(textureQueue, tex)
	}

	for len(textureQueue) > 0 {
		nextTexture := textureQueue[0]
		textureQueue = textureQueue[1:]

		// If we've already unpacked this texture, move on
		if _, ok := assets.Textures[nextTexture]; ok {
			continue
		}

		// If there is no namespace on the blockname, we assume the default of "minecraft"
		resourceName := strings.Split(nextTexture, ":")
		var namespace, blockName string
		if len(resourceName) == 1 {
			namespace = "minecraft"
			blockName = resourceName[0]
		} else {
			namespace = resourceName[0]
			blockName = resourceName[1]
		}

		// TODO: we assume for simplicity that the filetype is "png", and all minecraft textures are png. Do we need to support other image formats?
		filePath := fmt.Sprintf("assets/%s/textures/%s.png", namespace, blockName)

		if file, ok := fileMap[filePath]; ok {
			// First, load the file data into memory. Then create a raylib image from that byte slice.

			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			imageData, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			rlImage := rl.LoadImageFromMemory(".png", imageData, int32(len(imageData)))

			fmt.Printf("rlImage: %+v\n", rlImage)
			assets.Textures[nextTexture] = rl.LoadTextureFromImage(rlImage)
		} else {
			fmt.Printf("no resource \"%s\" found at path %s\n", nextTexture, filePath)
		}
	}

	fmt.Printf("textures:\n%+v\n", assets.Textures)

	return &assets, nil
}
