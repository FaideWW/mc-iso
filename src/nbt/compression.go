package nbt

import (
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
)

const (
	Uncompressed = iota
	Gzip
	Zlib
)

type DecompressibleReader = interface {
	io.ReaderAt
	io.Reader
}

// Peeks the first byte in the reader to check for
// compression headers. Returns 0 if uncompressed, or a
// constant referring to the likely detected compression
// library used (currently supports gzip and zlib)
func CheckCompression(r io.ReaderAt) (int, error) {
	var buf [1]byte
	n, err := r.ReadAt(buf[:], 0)
	if err != nil {
		return -1, err
	}
	if n != 1 {
		return -1, fmt.Errorf("read %d bytes when checking compression", n)
	}

	switch buf[0] {
	case 0x1f: // gzip magic header
		return Gzip, nil
	case 0x78: // zlib magic header
		return Zlib, nil
	case TAG_Compound: // likely uncompressed
		return Uncompressed, nil
	default: // either not an NBT file, or an unrecognized compression format
		return -1, fmt.Errorf("unrecognized first byte %#02x - either not an NBT file, or an unsupported compression format", buf[0])
	}
}

func Decompress(r DecompressibleReader) (io.Reader, error) {
	isCompressed, err := CheckCompression(r)
	if err != nil {
		return nil, err
	}

	var decompressed io.Reader

	switch isCompressed {
	case Uncompressed:
		decompressed = r
	case Gzip:
		decompressed, err = gzip.NewReader(r)
	case Zlib:
		decompressed, err = zlib.NewReader(r)
	}
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
