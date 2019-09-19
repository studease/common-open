package flv

// Tag formats the given data into an FLV tag
func Tag(typ byte, timestamp uint32, data []byte) []byte {
	size := len(data)
	backPointer := size + 11
	buf := make([]byte, backPointer+4)
	i := 0

	// header
	copy(buf[i:11], []byte{
		typ,
		byte(size >> 16), byte(size >> 8), byte(size),
		byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp), byte(timestamp >> 24),
		0x00, 0x00, 0x00,
	})
	i += 11

	// data
	copy(buf[i:i+size], data)
	i += size

	// size
	copy(buf[i:], []byte{
		byte(backPointer >> 24), byte(backPointer >> 16), byte(backPointer >> 8), byte(backPointer),
	})
	i += 4

	return buf
}
