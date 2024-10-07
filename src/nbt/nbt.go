package nbt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
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

// NBT decoder and unmarshaler.
// Heavily inspired by Tnze/go-mc and encoding/json

type NBTReader = interface {
	io.ByteReader
	io.Reader
}

// NBTByteReader implements ReadByte for io.Readers that aren't also io.ByteReaders
type NBTByteReader struct{ io.Reader }

func (r NBTByteReader) ReadByte() (byte, error) {
	var b [1]byte
	n, err := r.Read(b[:])
	if n == 1 {
		return b[0], nil
	}
	return 0, err
}

// NBTUnmarshaler allows callers to implement custom unmarshaling
// logic
type NBTUnmarshaler interface {
	UnmarshalNBT(tagType byte, r NBTReader) error
}

type NBTDecoder struct {
	r NBTReader
}

func NewDecoder(r io.Reader) *NBTDecoder {
	d := &NBTDecoder{}
	if nbtR, ok := r.(NBTReader); ok {
		d.r = nbtR
	} else {
		d.r = NBTByteReader{r}
	}
	return d
}

// Decodes an NBT value from the decoder's reader into v.
func (d *NBTDecoder) Decode(v any) (string, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return "", errors.New("nbt: non-pointer passed to Decode")
	}

	// Read the top-level tag header (usually this is TAG_Compound)
	tagType, tagName, err := d.ReadTagHeader()
	if err != nil {
		return tagName, err
	}

	err = d.unmarshal(val, tagType)
	if err != nil {
		return tagName, fmt.Errorf("nbt: failed to decode tag %q: %w", tagName, err)
	}
	return tagName, nil
}

// Reads the tag body from the decoder's reader (determined by tagType), and
// unmarshals it into v if possible
func (d *NBTDecoder) unmarshal(val reflect.Value, tagType byte) error {
	// ensure we have a settable pointer (or an unmarshaler)
	u, t, val := indirect(val, tagType == TAG_End)
	if u != nil {
		return u.UnmarshalNBT(tagType, d.r)
	}

	switch tagType {
	case TAG_End:
		return errors.New("unexpected TAG_End")
	case TAG_Byte:
		byte, err := d.ReadInt8()
		if err != nil {
			return err
		}
		switch vk := val.Kind(); vk {
		case reflect.Bool:
			val.SetBool(byte != 0)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(byte))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val.SetUint(uint64(byte))
		case reflect.Interface:
			val.Set(reflect.ValueOf(byte))
		default:
			return fmt.Errorf("can't unmarshal TAG_Byte into go type %q", vk.String())
		}
	case TAG_Short:
		value, err := d.ReadInt16()
		if err != nil {
			return err
		}
		switch vk := val.Kind(); vk {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(value))
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val.SetUint(uint64(value))
		case reflect.Interface:
			val.Set(reflect.ValueOf(value))
		default:
			return fmt.Errorf("can't unmarshal TAG_Short into go type %q", vk.String())
		}
	case TAG_Int:
		value, err := d.ReadInt32()
		if err != nil {
			return err
		}
		switch vk := val.Kind(); vk {
		case reflect.Int, reflect.Int32, reflect.Int64:
			val.SetInt(int64(value))
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			val.SetUint(uint64(value))
		case reflect.Interface:
			val.Set(reflect.ValueOf(value))
		default:
			return fmt.Errorf("can't unmarshal TAG_Int into go type %q", vk.String())
		}
	case TAG_Long:
		value, err := d.ReadInt64()
		if err != nil {
			return err
		}
		switch vk := val.Kind(); vk {
		case reflect.Int, reflect.Int64:
			val.SetInt(int64(value))
		case reflect.Uint, reflect.Uint64:
			val.SetUint(uint64(value))
		case reflect.Interface:
			val.Set(reflect.ValueOf(value))
		default:
			return fmt.Errorf("can't unmarshal TAG_Long into go type %q", vk.String())
		}
	case TAG_Float:
		vInt, err := d.ReadInt32()
		if err != nil {
			return err
		}
		value := math.Float32frombits(uint32(vInt))
		switch vk := val.Kind(); vk {
		case reflect.Float32, reflect.Float64:
			val.SetFloat(float64(value))
		case reflect.Interface:
			val.Set(reflect.ValueOf(value))
		default:
			return fmt.Errorf("can't unmarshal TAG_Float into go type %q", vk.String())
		}
	case TAG_Double:
		vInt, err := d.ReadInt64()
		if err != nil {
			return err
		}
		value := math.Float64frombits(uint64(vInt))
		switch vk := val.Kind(); vk {
		case reflect.Float64:
			val.SetFloat(float64(value))
		case reflect.Interface:
			val.Set(reflect.ValueOf(value))
		default:
			return fmt.Errorf("can't unmarshal TAG_Double into go type %q", vk.String())
		}
	case TAG_Byte_Array:
		arrayLen, err := d.ReadInt32()
		if err != nil {
			return err
		}

		vt := val.Type()
		vk := vt.Kind()
		if vk == reflect.Interface {
			vt = reflect.TypeOf([]byte{})
		} else if vk == reflect.Array && vt.Len() != int(arrayLen) {
			return fmt.Errorf("can't unmarshal TAG_Byte_Array into %q - length does not match", vt.String())
		} else if vk != reflect.Slice && vk != reflect.Array {
			return fmt.Errorf("can't unmarshal TAG_Byte_Array into go type %q", vt.String())
		} else if ek := vt.Elem().Kind(); ek != reflect.Uint8 && ek != reflect.Int8 {
			return fmt.Errorf("can't unmarshal TAG_Byte_Array into go type %q", vt.String())
		}

		// if we're working with an array, we can write directly to it. if we're
		// working with a slice, we need to allocate a new one with the correct
		// size
		buf := val
		if vk != reflect.Array {
			buf = reflect.MakeSlice(vt, int(arrayLen), int(arrayLen))
		}
		for i := 0; i < int(arrayLen); i++ {
			byte, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			buf.Index(i).Set(reflect.ValueOf(byte))
		}

		if vk != reflect.Array {
			val.Set(buf)
		}

	case TAG_String:
		str, err := d.ReadString()
		if err != nil {
			return err
		}
		if t != nil {
			err := t.UnmarshalText([]byte(str))
			if err != nil {
				return err
			}
		}

		switch vk := val.Kind(); vk {
		case reflect.String:
			val.SetString(str)
		case reflect.Interface:
			val.Set(reflect.ValueOf(str))
		default:
			return fmt.Errorf("can't unmarshal TAG_String into go type %q", vk.String())
		}
	case TAG_List:
		listType, err := d.r.ReadByte()
		if err != nil {
			return err
		}

		listLen, err := d.ReadInt32()
		if err != nil {
			return err
		}
		if listLen < 0 {
			return errors.New("can't unmarshal TAG_List with negative length")
		}

		var buf reflect.Value
		vk := val.Kind()
		switch vk {
		case reflect.Interface:
			buf = reflect.ValueOf(make([]any, listLen))
		case reflect.Slice:
			buf = reflect.MakeSlice(val.Type(), int(listLen), int(listLen))
		case reflect.Array:
			if arrLen := val.Len(); arrLen < int(listLen) {
				return fmt.Errorf("can't unmarshal TAG_List of len %d into array of len %d", listLen, arrLen)
			}
			buf = val
		default:
			return fmt.Errorf("can't unmarshal TAG_List into go type %q", vk.String())
		}

		for i := 0; i < int(listLen); i++ {
			if err := d.unmarshal(buf.Index(i), listType); err != nil {
				return err
			}
		}

		if vk != reflect.Array {
			val.Set(buf)
		}
	case TAG_Compound:
		u, ut, val := indirect(val, false)
		if u != nil {
			return u.UnmarshalNBT(tagType, d.r)
		}
		if ut != nil {
			return errors.New("can't unmarshal TAG_Compound into a string")
		}

		switch vk := val.Kind(); vk {
		case reflect.Struct:
			// parse the struct fields and their struct tags
			fields := cachedTypeFields(val.Type())
			for {
				fieldTagType, fieldTagName, err := d.ReadTagHeader()
				if err != nil {
					return err
				}
				if fieldTagType == TAG_End {
					break
				}

				f, ok := fields.byExactName[fieldTagName]
				if ok {
					val := val
					// if the struct embeds other structs, we need to walk down the tree
					// until we reach the actual location of the field we're trying to
					// set. f.index contains a path to traverse, where the values are the
					// index of the field at each level where the next level can be found
					for _, i := range f.index {
						if val.Kind() == reflect.Pointer {
							// if val points to an uninitialized pointer, initialize it.
							if val.IsNil() {
								// check if the field is an exported value
								if !val.CanSet() {
									return fmt.Errorf("cannot set embedded pointer to unexported struct: %v", val.Type().Elem())
								}
								val.Set(reflect.New(val.Type().Elem()))
							}
							val = val.Elem()
						}
						val = val.Field(i)
					}

					err = d.unmarshal(val, fieldTagType)
					if err != nil {
						// wrap error so we know where it's coming from
						return fmt.Errorf("failed to decode field %q in TAG_Compound: %w", fieldTagName, err)
					}
				} else {
					// fmt.Printf("no matching struct field found for tagname %q (type %#02x) - discarding\n", fieldTagName, fieldTagType)
					if err := d.ReadAndDiscardTag(fieldTagType); err != nil {
						// if we can't find a field to write the tag to, discard it
						return err
					}
				}
			}

		case reflect.Map:
			vt := val.Type()
			if vt.Key().Kind() != reflect.String {
				return fmt.Errorf("can't parse TagCompound as %q", vt.String())
			}
			if val.IsNil() {
				val.Set(reflect.MakeMap(vt))
			}
			for {
				fieldTagType, fieldTagName, err := d.ReadTagHeader()
				if err != nil {
					return err
				}
				if fieldTagType == TAG_End {
					break
				}
				v := reflect.New(vt.Elem())
				if err = d.unmarshal(v.Elem(), fieldTagType); err != nil {
					return fmt.Errorf("failed to decode field %q in TAG_Compound: %w", fieldTagName, err)
				}
				val.SetMapIndex(reflect.ValueOf(fieldTagName), v.Elem())
			}
		case reflect.Interface:
			buf := make(map[string]any)
			for {
				fieldTagType, fieldTagName, err := d.ReadTagHeader()
				if err != nil {
					return err
				}
				if fieldTagType == TAG_End {
					break
				}
				var value any
				if err = d.unmarshal(reflect.ValueOf(&value).Elem(), fieldTagType); err != nil {
					return fmt.Errorf("failed to decode field %q in TAG_Compound: %w", fieldTagName, err)
				}
				buf[fieldTagName] = value
			}
			val.Set(reflect.ValueOf(buf))
		default:
			return fmt.Errorf("can't unmarshal TAG_Compound into go type %q", vk.String())
		}
	case TAG_Int_Array:
		arrayLen, err := d.ReadInt32()
		if err != nil {
			return err
		}

		vt := val.Type()
		vk := vt.Kind()
		if vk == reflect.Interface {
			vt = reflect.TypeOf([]int32{})
		} else if vk == reflect.Array && vt.Len() != int(arrayLen) {
			return fmt.Errorf("can't unmarshal TAG_Int_Array into %q - length does not match", vt.String())
		} else if vk != reflect.Slice && vk != reflect.Array {
			return fmt.Errorf("can't unmarshal TAG_Int_Array into go type %q", vt.String())
		} else if ek := vt.Elem().Kind(); ek != reflect.Int && ek != reflect.Int32 {
			return fmt.Errorf("can't unmarshal TAG_Int_Array into go type %q", vt.String())
		}

		// if we're working with an array, we can write directly to it. if we're
		// working with a slice, we need to allocate a new one with the correct
		// size
		buf := val
		if vk != reflect.Array {
			buf = reflect.MakeSlice(vt, int(arrayLen), int(arrayLen))
		}
		for i := 0; i < int(arrayLen); i++ {
			value, err := d.ReadInt32()
			if err != nil {
				return err
			}
			buf.Index(i).SetInt(int64(value))
		}

		if vk != reflect.Array {
			val.Set(buf)
		}

	case TAG_Long_Array:
		arrayLen, err := d.ReadInt32()
		if err != nil {
			return err
		}

		vt := val.Type()
		vk := vt.Kind()
		if vk == reflect.Interface {
			vt = reflect.TypeOf([]int64{})
		} else if vk == reflect.Array && vt.Len() != int(arrayLen) {
			return fmt.Errorf("can't unmarshal TAG_Long_Array into %q - length does not match", vt.String())
		} else if vk != reflect.Slice && vk != reflect.Array {
			return fmt.Errorf("can't unmarshal TAG_Long_Array into go type %q", vt.String())
		} else if ek := vt.Elem().Kind(); ek != reflect.Int64 {
			return fmt.Errorf("can't unmarshal TAG_Long_Array into go type %q", vt.String())
		}

		// if we're working with an array, we can write directly to it. if we're
		// working with a slice, we need to allocate a new one with the correct
		// size
		buf := val
		if vk == reflect.Slice {
			buf = reflect.MakeSlice(vt, int(arrayLen), int(arrayLen))
		}
		for i := 0; i < int(arrayLen); i++ {
			value, err := d.ReadInt64()
			if err != nil {
				return err
			}
			buf.Index(i).SetInt(value)
		}

		if vt.Kind() == reflect.Slice {
			val.Set(buf)
		}

	default:
		return fmt.Errorf("can't unmarshal unknown tag type %#02x", tagType)
	}
	return nil
}

// Read primitives

func (d *NBTDecoder) ReadAndDiscardTag(tagType byte) error {
	var buf [8]byte
	switch tagType {
	case TAG_Byte:
		_, err := d.r.ReadByte()
		return err
	case TAG_Short:
		_, err := io.ReadFull(d.r, buf[:2])
		return err
	case TAG_Int, TAG_Float:
		_, err := io.ReadFull(d.r, buf[:4])
		return err
	case TAG_Long, TAG_Double:
		_, err := io.ReadFull(d.r, buf[:8])
		return err
	case TAG_Byte_Array:
		length, err := d.ReadInt32()
		if err != nil {
			return err
		}

		if _, err := io.CopyN(io.Discard, d.r, int64(length)); err != nil {
			return err
		}
	case TAG_String:
		_, err := d.ReadString()
		return err
	case TAG_List:
		elemType, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		length, err := d.ReadInt32()
		if err != nil {
			return err
		}
		for i := 0; i < int(length); i++ {
			if err := d.ReadAndDiscardTag(elemType); err != nil {
				return err
			}
		}
	case TAG_Compound:
		for {
			innerTagType, _, err := d.ReadTagHeader()
			if err != nil {
				return err
			}
			if innerTagType == TAG_End {
				break
			}
			err = d.ReadAndDiscardTag(innerTagType)
			if err != nil {
				return err
			}
		}

	case TAG_Int_Array:
		length, err := d.ReadInt32()
		if err != nil {
			return err
		}

		if _, err := io.CopyN(io.Discard, d.r, int64(length)*4); err != nil {
			return err
		}

	case TAG_Long_Array:
		length, err := d.ReadInt32()
		if err != nil {
			return err
		}

		if _, err := io.CopyN(io.Discard, d.r, int64(length)*8); err != nil {
			return err
		}

	}
	return nil
}

func (d *NBTDecoder) ReadTagHeader() (byte, string, error) {
	var tagType byte
	var tagName string
	var err error
	tagType, err = d.r.ReadByte()
	if err != nil {
		return tagType, tagName, err
	}

	// Check if we're reading a compressed stream
	switch tagType {
	case 0x1f: // gzip magic header
		err = fmt.Errorf("nbt: unknown tag %#02x - possibly reading a compressed gzip stream", tagType)
	case 0x78: // zlib magic header
		err = fmt.Errorf("nbt: unknown tag %#02x - possibly reading a compressed zlib stream", tagType)
	case TAG_End:
		// TAG_End does not have a tagname, so do nothing
	default:
		tagName, err = d.ReadString()
	}

	return tagType, tagName, err
}

func (d *NBTDecoder) ReadInt8() (int8, error) {
	byte, err := d.r.ReadByte()
	return int8(byte), err
}
func (d *NBTDecoder) ReadInt16() (int16, error) {
	var v int16
	err := binary.Read(d.r, binary.BigEndian, &v)
	return v, err
}
func (d *NBTDecoder) ReadInt32() (int32, error) {
	var v int32
	err := binary.Read(d.r, binary.BigEndian, &v)
	return v, err
}
func (d *NBTDecoder) ReadInt64() (int64, error) {
	var v int64
	err := binary.Read(d.r, binary.BigEndian, &v)
	return v, err
}
func (d *NBTDecoder) ReadString() (string, error) {
	strLen, err := d.ReadInt16()
	if err != nil {
		return "", err
	}

	if strLen < 0 {
		return "", errors.New("string has negative length")
	}

	var str string
	if strLen > 0 {
		buffer := make([]byte, strLen)
		_, err = io.ReadFull(d.r, buffer)
		str = string(buffer)
	}
	return str, err
}
