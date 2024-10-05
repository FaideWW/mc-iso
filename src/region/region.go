package region

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/faideww/mc-iso/src/nbt"
)

type ChunkLocation struct {
	// location of the chunk from the start of the file (measured in 4KiB sectors)
	offset uint32
	// length of the chunk data (also measured in 4KiB sectors)
	size byte
}

// A region describes a group of 32x32 chunks
type Region struct {
	// location of each chunk in the file
	locTable [1024]ChunkLocation

	// time of last modification for each chunk
	timestampTable [1024]uint32

	Chunks [1024]Chunk
}

func NewRegion(r io.ReadSeeker) (Region, error) {
	var region Region

	// First 4096 bytes are the location table
	buf := make([]byte, 4096)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return region, err
	}
	if n != 4096 {
		return region, fmt.Errorf("failed to read location table; only read %d bytes (expected 4096)", n)
	}

	for i := 0; i < 1024; i++ {
		offset := i * 4

		region.locTable[i].offset = uint32(buf[offset])<<16 | uint32(buf[offset+1])<<8 | uint32(buf[offset+2])
		// binary.BigEndian.Uint32(buf[offset : offset+2])
		region.locTable[i].size = buf[offset+3]

		// fmt.Printf("chunk %d is at loc %d (size %d)\n", i, region.locTable[i].offset, region.locTable[i].size)
	}

	// Next 4096 bytes are the timestamp table
	err = binary.Read(r, binary.BigEndian, &region.timestampTable)
	if err != nil {
		return region, err
	}

	// Finally, the chunk payload.
	// Each chunk begins with a 5-byte header:
	// - 4 bytes describing the (unpadded) length of the chunk data in bytes
	// - 1 byte describing the compression type
	for i := 0; i < 1024; i++ {
		if region.locTable[i].offset == 0 && region.locTable[i].size == 0 {
			// if both offset and size are 0, there is no chunk in this location
			continue
		}

		// seek to the start of the chunk
		r.Seek(int64(region.locTable[i].offset*4096), io.SeekStart)

		var chunkLen int32
		var compression byte

		if err := binary.Read(r, binary.BigEndian, &chunkLen); err != nil {
			return region, err
		}
		if err := binary.Read(r, binary.BigEndian, &compression); err != nil {
			return region, err
		}

		var decompressed io.Reader
		switch compression {
		case 0x01:
			decompressed, err = gzip.NewReader(r)
			if err != nil {
				return region, err
			}
		case 0x02:
			decompressed, err = zlib.NewReader(r)
			if err != nil {
				return region, err
			}
		case 0x03:
			decompressed = r
		default:
			return region, fmt.Errorf("unrecognized compression scheme %#02x", compression)
		}

		var c Chunk
		_, err = nbt.NewDecoder(decompressed).Decode(&c)
		if err != nil {
			return region, err
		}

		c.Loaded = true

		region.Chunks[i] = c
	}
	return region, nil
}
