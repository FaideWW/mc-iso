package region

type Chunk struct {
	DataVersion int       `nbt:"DataVersion"`
	XPos        int       `nbt:"xPos"`
	ZPos        int       `nbt:"zPos"`
	YPos        int       `nbt:"yPos"`
	Status      string    `nbt:"Status"`
	LastUpdate  int64     `nbt:"LastUpdate"`
	Sections    []Section `nbt:"sections"`
}

type Section struct {
	Y           uint8      `nbt:"Y"`
	BlockStates Palette    `nbt:"block_states"`
	Biomes      Palette    `nbt:"biomes"`
	BlockLight  [2048]byte `nbt:"BlockLight"`
	SkyLight    [2048]byte `nbt:"SkyLight"`
}

type Palette struct {
	Palette []PaletteData `nbt:"palette"`
	Data    []int64       `nbt:"data"`
}

type PaletteData struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties"`
}

// TODO: do we need this?
type BlockEntity interface{}
