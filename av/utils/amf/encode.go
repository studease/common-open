package amf

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
)

// AppendBytes into buffer
func AppendBytes(b *bytes.Buffer, data []byte) (int, error) {
	return b.Write(data)
}

// AppendInt8 into buffer
func AppendInt8(b *bytes.Buffer, n int8) (int, error) {
	return 1, b.WriteByte(byte(n))
}

// AppendInt16 into buffer
func AppendInt16(b *bytes.Buffer, n int16, littleEndian bool) (int, error) {
	var (
		order binary.ByteOrder
	)

	if littleEndian {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	err := binary.Write(b, order, &n)
	if err != nil {
		return 0, err
	}

	return 2, nil
}

// AppendInt32 into buffer
func AppendInt32(b *bytes.Buffer, n int32, littleEndian bool) (int, error) {
	var (
		order binary.ByteOrder
	)

	if littleEndian {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	err := binary.Write(b, order, &n)
	if err != nil {
		return 0, err
	}

	return 4, nil
}

// AppendUint8 into buffer
func AppendUint8(b *bytes.Buffer, n uint8) (int, error) {
	return 1, b.WriteByte(byte(n))
}

// AppendUint16 into buffer
func AppendUint16(b *bytes.Buffer, n uint16, littleEndian bool) (int, error) {
	var (
		order binary.ByteOrder
	)

	if littleEndian {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	err := binary.Write(b, order, &n)
	if err != nil {
		return 0, err
	}

	return 2, nil
}

// AppendUint32 into buffer
func AppendUint32(b *bytes.Buffer, n uint32, littleEndian bool) (int, error) {
	var (
		order binary.ByteOrder
	)

	if littleEndian {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	err := binary.Write(b, order, &n)
	if err != nil {
		return 0, err
	}

	return 4, nil
}

// Encode an AMF Value into buffer
func Encode(b *bytes.Buffer, v *Value) (int, error) {
	switch v.Type {
	case DOUBLE:
		return EncodeDouble(b, v.value.(float64))

	case BOOLEAN:
		return EncodeBoolean(b, v.value.(bool))

	case STRING:
		fallthrough
	case LONG_STRING:
		return EncodeString(b, v.value.(string))

	case OBJECT:
		return EncodeObject(b, v)

	case ECMA_ARRAY:
		return EncodeECMAArray(b, v)

	case STRICT_ARRAY:
		return EncodeStrictArray(b, v)

	case DATE:
		return EncodeDate(b, v.value.(float64), v.offset)

	case NULL:
		return EncodeNull(b)

	case UNDEFINED:
		return EncodeUndefined(b)

	default:
		panic(fmt.Errorf("unrecognized AMF type 0x%02X", v.Type))
	}
}

// EncodeDouble writes a float64 into buffer
func EncodeDouble(b *bytes.Buffer, n float64) (int, error) {
	i := 0

	if err := b.WriteByte(DOUBLE); err != nil {
		return i, err
	}

	i++

	if err := binary.Write(b, binary.BigEndian, &n); err != nil {
		return i, err
	}

	i += 8

	return i, nil
}

// EncodeBoolean writes a bool into buffer
func EncodeBoolean(b *bytes.Buffer, v bool) (int, error) {
	n := byte(0)
	i := 0

	if v {
		n = 1
	}

	if err := b.WriteByte(BOOLEAN); err != nil {
		return i, err
	}

	i++

	if err := b.WriteByte(n); err != nil {
		return i, err
	}

	i++

	return i, nil
}

// EncodeString writes a string into buffer
func EncodeString(b *bytes.Buffer, s string) (int, error) {
	x := len(s)
	i := 0

	if x >= 0xFFFF {
		return encodeLongString(b, s)
	}

	if err := b.WriteByte(STRING); err != nil {
		return i, err
	}

	i++

	m := uint16(x)
	if err := binary.Write(b, binary.BigEndian, &m); err != nil {
		return i, err
	}

	i += 2

	n, err := b.Write([]byte(s))
	if err != nil {
		return i, err
	}

	i += n

	return i, nil
}

func encodeLongString(b *bytes.Buffer, s string) (int, error) {
	x := len(s)
	i := 0

	if err := b.WriteByte(LONG_STRING); err != nil {
		return i, err
	}

	i++

	t := uint32(x)
	if err := binary.Write(b, binary.BigEndian, &t); err != nil {
		return i, err
	}

	i += 4

	n, err := b.Write([]byte(s))
	if err != nil {
		return i, err
	}

	i += n

	return i, nil
}

// EncodeDate writes a time into buffer
func EncodeDate(b *bytes.Buffer, timestamp float64, offset uint16) (int, error) {
	i := 0

	if err := b.WriteByte(byte(DATE)); err != nil {
		return i, err
	}

	i++

	if err := binary.Write(b, binary.BigEndian, &timestamp); err != nil {
		return i, err
	}

	i += 8

	if err := binary.Write(b, binary.BigEndian, &offset); err != nil {
		return i, err
	}

	i += 2

	return i, nil
}

// EncodeObject writes an object into buffer
func EncodeObject(b *bytes.Buffer, v *Value) (int, error) {
	if v.Type != OBJECT {
		panic("type should be OBJECT")
	}

	i := 0

	if err := b.WriteByte(OBJECT); err != nil {
		return i, err
	}

	i++

	n, err := encodeProperties(b, v.value.(*list.List))
	if err != nil {
		return i, err
	}

	i += n

	n, err = EndOfObject(b)
	if err != nil {
		return i, err
	}

	i += n

	return i, nil
}

func encodeProperties(b *bytes.Buffer, l *list.List) (int, error) {
	i := 0

	for e := l.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Value)

		if t := uint16(len(v.Key)); t > 0 {
			err := binary.Write(b, binary.BigEndian, &t)
			if err != nil {
				return i, err
			}

			i += 2

			n, err := b.Write([]byte(v.Key))
			if err != nil {
				return i, err
			}

			i += n
		}

		n, err := Encode(b, v)
		if err != nil {
			return i, err
		}

		i += n
	}

	return i, nil
}

// EndOfObject writes an END_OF_OBJECT block (0x00 0x00 0x09) into buffer
func EndOfObject(b *bytes.Buffer) (int, error) {
	return b.Write([]byte{0x00, 0x00, 0x09})
}

// EncodeECMAArray writes an ecma array into buffer
func EncodeECMAArray(b *bytes.Buffer, v *Value) (int, error) {
	i := 0

	if err := b.WriteByte(ECMA_ARRAY); err != nil {
		return i, err
	}

	i++

	t := uint32(v.value.(*list.List).Len())
	if err := binary.Write(b, binary.BigEndian, &t); err != nil {
		return i, err
	}

	i += 4

	n, err := encodeProperties(b, v.value.(*list.List))
	if err != nil {
		return i, err
	}

	i += n

	n, err = EndOfObject(b)
	if err != nil {
		return i, err
	}

	i += n

	return i, nil
}

// EncodeStrictArray writes a strict array into buffer
func EncodeStrictArray(b *bytes.Buffer, v *Value) (int, error) {
	i := 0

	if err := b.WriteByte(STRICT_ARRAY); err != nil {
		return i, err
	}

	i++

	t := uint32(v.value.(*list.List).Len())
	if err := binary.Write(b, binary.BigEndian, &t); err != nil {
		return i, err
	}

	i += 4

	n, err := encodeProperties(b, v.value.(*list.List))
	if err != nil {
		return i, err
	}

	i += n

	return i, nil
}

// EncodeNull writes a null into buffer
func EncodeNull(b *bytes.Buffer) (int, error) {
	i := 0

	if err := b.WriteByte(NULL); err != nil {
		return i, err
	}

	return i, nil
}

// EncodeUndefined writes a undefined into buffer
func EncodeUndefined(b *bytes.Buffer) (int, error) {
	i := 0

	if err := b.WriteByte(UNDEFINED); err != nil {
		return i, err
	}

	return i, nil
}
