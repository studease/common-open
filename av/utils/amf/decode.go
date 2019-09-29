package amf

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"math"
)

// Decode decodes a typed AMF Value
func Decode(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 1 {
		return 0, fmt.Errorf("data not enough while decoding AMF Value")
	}

	i := 0

	v.Type = data[i]
	i++

	var (
		n   int
		err error
	)

	switch v.Type {
	case DOUBLE:
		n, err = DecodeDouble(v, data[i:])
	case BOOLEAN:
		n, err = DecodeBoolean(v, data[i:])
	case STRING:
		n, err = DecodeString(v, data[i:])
	case OBJECT:
		n, err = DecodeObject(v, data[i:])
	case NULL:
	case UNDEFINED:
	case ECMA_ARRAY:
		n, err = DecodeECMAArray(v, data[i:])
	case END_OF_OBJECT:
		n, err = DecodeEndOfObject(v, data[i:])
	case STRICT_ARRAY:
		n, err = DecodeStrictArray(v, data[i:])
	case DATE:
		n, err = DecodeDate(v, data[i:])
	case LONG_STRING:
		n, err = DecodeLongString(v, data[i:])
	default:
		return i, fmt.Errorf("unrecognized amf type: 0x%02X", v.Type)
	}

	i += n

	return i, err
}

// DecodeDouble decodes an AMF double from data
func DecodeDouble(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 8 {
		return 0, fmt.Errorf("data not enough while decoding AMF double")
	}

	i := 0

	n := binary.BigEndian.Uint64(data[i : i+8])
	v.value = math.Float64frombits(n)
	i += 8

	return i, nil
}

// DecodeBoolean decodes an AMF boolean from data
func DecodeBoolean(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 1 {
		return 0, fmt.Errorf("data not enough while decoding AMF boolean")
	}

	i := 0

	v.value = data[i] > 0
	i++

	return i, nil
}

// DecodeString decodes an AMF string from data
func DecodeString(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 2 {
		return 0, fmt.Errorf("data not enough while decoding AMF string")
	}

	i := 0

	n := binary.BigEndian.Uint16(data[i : i+2])
	i += 2

	if n > 0 {
		v.value = string(data[i : i+int(n)])
		i += int(n)
	} else {
		v.value = ""
	}

	return i, nil
}

// DecodeLongString decodes an AMF long string from data
func DecodeLongString(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 4 {
		return 0, fmt.Errorf("data not enough while decoding AMF long string")
	}

	i := 0

	n := binary.BigEndian.Uint32(data[i : i+4])
	i += 4

	if n > 0 {
		v.value = string(data[i : i+int(n)])
		i += int(n)
	}

	return i, nil
}

// DecodeObject decodes an AMF object from data
func DecodeObject(o *Value, data []byte) (int, error) {
	size := len(data)
	if size < 3 {
		return 0, fmt.Errorf("data not enough while decoding AMF object")
	}

	i := 0

	o.value = list.New()
	o.table = make(map[string]*Value)

	for i < size {
		v := NewValue(DOUBLE)

		n, err := DecodeString(v, data[i:])
		if err != nil {
			return i, err
		}

		v.Key = v.String()
		i += n

		n, err = Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n

		if v.Type == END_OF_OBJECT {
			break
		}

		o.value.(*list.List).PushBack(v)
		o.table[v.Key] = v
	}

	return i, nil
}

// DecodeECMAArray decodes an AMF ecma array from data
func DecodeECMAArray(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 4 {
		return 0, fmt.Errorf("data not enough while decoding AMF ECMA array")
	}

	i := 4 // Don't trust array length field

	n, err := DecodeObject(v, data[i:])
	if err != nil {
		return i, err
	}

	i += n

	return i, nil
}

// DecodeStrictArray decodes an AMF strict array from data
func DecodeStrictArray(o *Value, data []byte) (int, error) {
	size := len(data)
	if size < 4 {
		return 0, fmt.Errorf("data not enough while decoding AMF strict array")
	}

	i := 0

	length := binary.BigEndian.Uint32(data[i : i+4])
	i += 4

	o.value = list.New()

	for j := uint32(0); j < length && i < size; j++ {
		v := NewValue(DOUBLE)

		n, err := Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n

		o.value.(*list.List).PushBack(v)
	}

	if i+3 <= size && data[i] == 0 && data[i+1] == 0 && data[i+2] == byte(END_OF_OBJECT) {
		o.Ended = true
		i += 3
	}

	return i, nil
}

// DecodeDate decodes an AMF date from data
func DecodeDate(v *Value, data []byte) (int, error) {
	size := len(data)
	if size < 10 {
		return 0, fmt.Errorf("data not enough while decoding AMF date")
	}

	i := 0

	n := binary.BigEndian.Uint64(data[i : i+8])
	v.value = math.Float64frombits(n)
	i += 8

	v.offset = binary.BigEndian.Uint16(data[i : i+2])
	i += 2

	return i, nil
}

// DecodeEndOfObject decodes an AMF undefined from data
func DecodeEndOfObject(v *Value, data []byte) (int, error) {
	v.Ended = true
	return 0, nil
}
