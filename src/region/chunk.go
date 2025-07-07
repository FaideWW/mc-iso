package region

type Chunk struct {
	Loaded      bool
	DataVersion int       `nbt:"DataVersion"`
	XPos        int32     `nbt:"xPos"`
	ZPos        int32     `nbt:"zPos"`
	YPos        int32     `nbt:"yPos"`
	Status      string    `nbt:"Status"`
	LastUpdate  int64     `nbt:"LastUpdate"`
	Sections    []Section `nbt:"sections"`
}
