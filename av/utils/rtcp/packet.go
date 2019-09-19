package rtcp

import (
	"fmt"

	"github.com/studease/common/av/utils"
)

// Packet types
const (
	TYPE_SR   byte = 200
	TYPE_RR   byte = 201
	TYPE_SDES byte = 202
	TYPE_BYE  byte = 203
	TYPE_APP  byte = 204
)

// SDES types
const (
	SDES_END   uint8 = 0
	SDES_CNAME uint8 = 1
	SDES_NAME  uint8 = 2
	SDES_EMAIL uint8 = 3
	SDES_PHONE uint8 = 4
	SDES_LOC   uint8 = 5
	SDES_TOOL  uint8 = 6
	SDES_NOTE  uint8 = 7
	SDES_PRIV  uint8 = 8
)

// Header of RTCP packet
type Header struct {
	V      byte   // 2 bits
	P      byte   // 1 bit
	RC     byte   // 5 bits
	PT     byte   // 1 byte
	Length uint16 // 2 bytes
}

// SR (Sender Report)
type SR struct {
	SSRC      uint32
	NTPSec    uint32
	NTPFrac   uint32
	Timestamp uint32
	PSent     uint32
	OSent     uint32
	RBs       []ReportBlock
}

// RR (Receiver Report)
type RR struct {
	SSRC uint32
	RBs  []ReportBlock
}

// ReportBlock in SR and RR
type ReportBlock struct {
	SSRC     uint32
	Fraction byte // 1 byte
	Lost     int  // 3 bytes
	LastSeq  uint32
	Jitter   uint32
	LSR      uint32
	DLSR     uint32
}

// SDES (Source Description)
type SDES struct {
	SRC   uint32
	Items []Item
}

// Item of SDES
type Item struct {
	Type   uint8
	Length uint8
	Data   []byte
}

// BYE (Goodbye)
type BYE struct {
	SRC []uint32
}

// Packet of RTCP
type Packet struct {
	Header
	SR
	RR
	SDES
	BYE
	Data []byte
}

// Init this class
func (me *Packet) Init() *Packet {
	return me
}

// Parse the given data as an RTCP packet
func (me *Packet) Parse(data []byte) (int, error) {
	var (
		size = len(data)
		b    utils.BitStream
	)

	if size < 4 {
		return 0, fmt.Errorf("data not enough: %d/%d", size, 4)
	}

	b.Init(data)

	me.V = byte(b.ReadBits(2))
	me.P = byte(b.ReadBits(1))
	me.RC = byte(b.ReadBits(5))
	me.PT = byte(b.ReadBits(8))
	me.Length = uint16(b.ReadBits(16))

	switch me.PT {
	case TYPE_SR:
		me.SR.SSRC = uint32(b.ReadBitsLong(32))
		me.SR.NTPSec = uint32(b.ReadBitsLong(32))
		me.SR.NTPFrac = uint32(b.ReadBitsLong(32))
		me.SR.Timestamp = uint32(b.ReadBitsLong(32))
		me.SR.PSent = uint32(b.ReadBitsLong(32))
		me.SR.OSent = uint32(b.ReadBitsLong(32))
		me.SR.RBs = make([]ReportBlock, me.RC)

		for i := 0; i < int(me.RC); i++ {
			me.SR.RBs[i].SSRC = uint32(b.ReadBitsLong(32))
			me.SR.RBs[i].Fraction = byte(b.ReadBits(8))
			me.SR.RBs[i].Lost = int(b.ReadBits(24))
			me.SR.RBs[i].LastSeq = uint32(b.ReadBitsLong(32))
			me.SR.RBs[i].Jitter = uint32(b.ReadBitsLong(32))
			me.SR.RBs[i].LSR = uint32(b.ReadBitsLong(32))
			me.SR.RBs[i].DLSR = uint32(b.ReadBitsLong(32))
		}

	case TYPE_RR:
		me.RR.SSRC = uint32(b.ReadBitsLong(32))
		me.RR.RBs = make([]ReportBlock, me.RC)

		for i := 0; i < int(me.RC); i++ {
			me.RR.RBs[i].SSRC = uint32(b.ReadBitsLong(32))
			me.RR.RBs[i].Fraction = byte(b.ReadBits(8))
			me.RR.RBs[i].Lost = int(b.ReadBits(24))
			me.RR.RBs[i].LastSeq = uint32(b.ReadBitsLong(32))
			me.RR.RBs[i].Jitter = uint32(b.ReadBitsLong(32))
			me.RR.RBs[i].LSR = uint32(b.ReadBitsLong(32))
			me.RR.RBs[i].DLSR = uint32(b.ReadBitsLong(32))
		}

	case TYPE_SDES:
		me.SDES.SRC = uint32(b.ReadBitsLong(32))
		me.SDES.Items = make([]Item, 0)

		for b.Left() >= 24 /* 3 bytes */ {
			item := new(Item)
			item.Type = uint8(b.ReadBits(8))

			if item.Type == SDES_END {
				break
			}

			item.Length = uint8(b.ReadBits(8))

			if b.Left() < int(item.Length)*8 {
				return b.Index / 8, fmt.Errorf("data not enough while parsing SDES item")
			}

			i := b.Index / 8
			b.SkipBits(int(item.Length) * 8)
			item.Data = data[i : i+int(item.Length)]

			me.SDES.Items = append(me.SDES.Items, *item)
		}

	case TYPE_BYE:
		for b.Left() >= 32 /* 4 bytes */ {
			me.BYE.SRC = append(me.BYE.SRC, uint32(b.ReadBitsLong(32)))
		}

	case TYPE_APP:

	default:
		return 4, fmt.Errorf("unrecognized rtcp packet type: %d", me.PT)
	}

	return b.Index / 8, nil
}
