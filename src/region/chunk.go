package region

type Chunk struct {
	Loaded      bool
	DataVersion int       `nbt:"DataVersion"`
	XPos        int       `nbt:"xPos"`
	ZPos        int       `nbt:"zPos"`
	YPos        int       `nbt:"yPos"`
	Status      string    `nbt:"Status"`
	LastUpdate  int64     `nbt:"LastUpdate"`
	Sections    []Section `nbt:"sections"`
}
