package rtp

import (
	"bytes"
)

// NAL types
const (
	NAL_UNIT   = 23
	NAL_STAP_A = 24
	NAL_STAP_B = 25
	NAL_MTAP16 = 26
	NAL_MTAP24 = 27
	NAL_FU_A   = 28
	NAL_FU_B   = 29
)

// Header of RTP packet
type Header struct {
	V         byte   // 2 bits
	P         byte   // 1 bit
	X         byte   // 1 bit
	CC        byte   // 4 bits
	M         byte   // 1 bit
	PT        byte   // 7 bits
	SN        uint16 // 2 bytes
	Timestamp uint32
	SSRC      uint32
	CSRC      []uint32
}

// Packet of RTP
type Packet struct {
	Header
	Payload []byte
}

// Init this class
func (me *Packet) Init() *Packet {
	me.V = Version
	return me
}

// Format returns the raw bytes of the packet
func (me *Packet) Format() []byte {
	var (
		b bytes.Buffer
	)

	b.Write([]byte{
		me.V<<6 | me.P<<5 | me.X<<4 | me.CC,
		me.M<<7 | me.PT,
		byte(me.SN >> 8), byte(me.SN),
		byte(me.Timestamp >> 24), byte(me.Timestamp >> 16), byte(me.Timestamp >> 8), byte(me.Timestamp),
		byte(me.SSRC >> 24), byte(me.SSRC >> 16), byte(me.SSRC >> 8), byte(me.SSRC),
	})

	for _, v := range me.CSRC {
		b.Write([]byte{
			byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v),
		})
	}

	b.Write(me.Payload)

	return b.Bytes()
}
