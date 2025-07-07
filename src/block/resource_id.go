package block

import "strings"

// TODO: do we need to support nested registry paths? (ie.
// "minecraft:block/cobblestone" vs. just "minecraft/cobblestone")
func DecodeResourceId(s string) (string, string) {
	// If there is no namespace on the blockname, we assume the default of "minecraft"
	resourceName := strings.Split(s, ":")
	var namespace, path string
	if len(resourceName) == 1 {
		namespace = "minecraft"
		path = resourceName[0]
	} else {
		namespace = resourceName[0]
		path = resourceName[1]
	}

	return namespace, path
}
