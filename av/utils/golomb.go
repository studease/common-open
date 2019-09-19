package utils

// Golomb defines methods for reading UE and SE
type Golomb struct {
	BitStream
}

// Init me class
func (me *Golomb) Init(b []byte) *Golomb {
	if me.BitStream.Init(b) == nil {
		return nil
	}

	return me
}

// ReadUE read an unsigned Exp-Golomb code
func (me *Golomb) ReadUE() uint32 {
	n := -1

	for i := uint32(0); i == 0 && me.Left() > 0; n++ {
		i = me.ReadBits(1)
	}

	return (1 << uint32(n)) - 1 + me.ReadBitsLong(n)
}

// ReadSE read an signed Exp-Golomb code
func (me *Golomb) ReadSE() int32 {
	u := me.ReadUE()
	if (u & 0x01) == 0 {
		return -int32(u >> 1)
	}

	return int32(u+1) >> 1
}
