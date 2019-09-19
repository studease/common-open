package utils

// Static constants
const (
	MIN_UINT32     uint32 = 0
	MAX_UINT32            = ^uint32(0)
	MAX_INT32             = int32(^uint32(0) >> 1)
	MIN_INT32             = ^MAX_INT32
	MIN_CACHE_BITS        = 25
)

// BitStream defines methods for showing and reading in bits
type BitStream struct {
	Index  int
	Bits   int
	Buffer []byte
}

// Init me class
func (me *BitStream) Init(b []byte) *BitStream {
	n := len(b)
	if n > int(MAX_INT32/8) {
		return nil
	}

	n *= 8
	if n >= int(MAX_INT32-7) || b == nil {
		return nil
	}

	me.Index = 0
	me.Bits = n
	me.Buffer = b

	return me
}

// ShowBits returns 1-25 bits
func (me *BitStream) ShowBits(n int) uint32 {
	i := me.Index >> 3
	x := uint32(0)

	if me.Index < me.Bits {
		x = uint32(me.Buffer[i]) << 24
	}

	if me.Index+8 < me.Bits {
		x |= uint32(me.Buffer[i+1]) << 16
	}

	if me.Index+16 < me.Bits {
		x |= uint32(me.Buffer[i+2]) << 8
	}

	if me.Index+24 < me.Bits {
		x |= uint32(me.Buffer[i+3])
	}

	cache := x << uint32(me.Index&7)

	return cache >> uint32(32-n)
}

// ShowBitsLong returns 0-32 bits
func (me *BitStream) ShowBitsLong(n int) uint32 {
	if n <= MIN_CACHE_BITS {
		return me.ShowBits(n)
	}

	return me.ShowBits(16)<<uint32(n-16) | me.ShowBits(n-16)
}

// ReadBits read 1-25 bits.
func (me *BitStream) ReadBits(n int) uint32 {
	tmp := me.ShowBits(n)
	me.Index += n
	return tmp
}

// ReadBitsLong read 0-32 bits.
func (me *BitStream) ReadBitsLong(n int) uint32 {
	tmp := me.ShowBitsLong(n)
	me.Index += n
	return tmp
}

// SkipBits skips represented bits
func (me *BitStream) SkipBits(n int) {
	me.Index += n
}

// Left returns bits remains
func (me *BitStream) Left() int {
	return me.Bits - me.Index
}
