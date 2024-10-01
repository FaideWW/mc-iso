package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	TAG_End = iota
	TAG_Byte
	TAG_Short
	TAG_Int
	TAG_Long
	TAG_Float
	TAG_Double
	TAG_Byte_Array
	TAG_String
	TAG_List
	TAG_Compound
	TAG_Int_Array
	TAG_Long_Array
)

func ParseNBT(in io.Reader) {
	r := bufio.NewReader(in)
	tagNameSizeBuf := make([]byte, 2)
	tagNameBuf := make([]byte, 256)
	payloadBuf := make([]byte, 16)
	for {
		tagNameSize := 0
		var tagName string
		tagType, err := r.ReadByte()

		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println(err)
			break
		}

		if int(tagType) != 0 {
			io.ReadFull(r, tagNameSizeBuf)
			tagNameSize = int(binary.BigEndian.Uint16(tagNameSizeBuf))

			// if the name buffer is too small, resize it until it's not
			for tagNameSize > len(tagNameBuf) {
				tagNameBuf = make([]byte, len(tagNameBuf)*2)
			}

			tagName = string(tagNameBuf[:tagNameSize])
		}

		switch int(tagType) {
		case TAG_End:
			fmt.Printf("TAG_End\n")
		case TAG_Byte:
			var byteVal byte
			binary.Read(r, binary.BigEndian, &byteVal)
			fmt.Printf("TAG_Byte name:'%s' payload: %d\n", tagName, byteVal)

		case TAG_Short:
			var shortVal int16
			binary.Read(r, binary.BigEndian, &shortVal)
			fmt.Printf("TAG_Short name:'%s' payload: %d\n", tagName, shortVal)

		case TAG_Int:
			var intVal int32
			binary.Read(r, binary.BigEndian, &intVal)
			fmt.Printf("TAG_Int name:'%s' payload: %d\n", tagName, intVal)
		case TAG_Long:
			var longVal int64
			binary.Read(r, binary.BigEndian, &longVal)
			fmt.Printf("TAG_Long name:'%s' payload: %d\n", tagName, longVal)
		case TAG_Float:
			var floatVal float32
			binary.Read(r, binary.BigEndian, &floatVal)
			fmt.Printf("TAG_Float name:'%s' payload: %d\n", tagName, floatVal)
		case TAG_Double:
			var doubleVal float64
			binary.Read(r, binary.BigEndian, &doubleVal)
			fmt.Printf("TAG_Double name:'%s' payload: %d\n", tagName, doubleVal)
		case TAG_Byte_Array:
			var arrayLen int32
			binary.Read(r, binary.BigEndian, &arrayLen)
			arr := make([]byte, arrayLen)
			binary.Read(r, binary.BigEndian, &arr)
			fmt.Printf("TAG_Byte_Array name:'%s' payload: %x\n", tagName, arr)
		case TAG_String:
			var strLen int16
			binary.Read(r, binary.BigEndian, &strLen)
			arr := make([]byte, strLen)
			binary.Read(r, binary.BigEndian, &arr)
			str := string(arr)
			fmt.Printf("TAG_Double name:'%s' payload: %s\n", tagName, str)
		case TAG_List:
			// TODO: this needs recursion
			var tagId byte
			binary.Read(r, binary.BigEndian, &tagId)
			var tagCount int32
			binary.Read(r, binary.BigEndian, &tagCount)
			arr := make([]byte, strLen)
			binary.Read(r, binary.BigEndian, &arr)
			str := string(arr)
			fmt.Printf("TAG_Double name:'%s' payload: %s\n", tagName, str)
		case TAG_Compound:
		case TAG_Int_Array:
		case TAG_Long_Array:
		}

		if err != nil {
			break
		}
	}
}
