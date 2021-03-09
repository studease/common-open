package message

import (
	"bytes"
	"fmt"
)

// Message types
const (
	SET_CHUNK_SIZE     byte = 0x01
	ABORT              byte = 0x02
	ACK                byte = 0x03
	USER_CONTROL       byte = 0x04
	ACK_WINDOW_SIZE    byte = 0x05
	BANDWIDTH          byte = 0x06
	EDGE               byte = 0x07
	AUDIO              byte = 0x08
	VIDEO              byte = 0x09
	DATA_AMF3          byte = 0x0F
	SHARED_OBJECT_AMF3 byte = 0x10
	COMMAND_AMF3       byte = 0x11
	DATA               byte = 0x12
	SHARED_OBJECT      byte = 0x13
	COMMAND            byte = 0x14
	AGGREGATE          byte = 0x16
)

// Chunk flags
const (
	FLAG_UNSET    uint8 = 0
	FLAG_ABSOLUTE uint8 = 1
	FLAG_DELTA    uint8 = 2
)

// IMessage defines basic rtmp message methods
type IMessage interface {
	Parse(data []byte) (int, error)
}

// Basic chunk header 1
// +-+-+-+-+-+-+-+-+
// |0 1 2 3 4 5 6 7|
// +-+-+-+-+-+-+-+-+
// |fmt|   cs id   |
// +-+-+-+-+-+-+-+-+

// Basic chunk header 2
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |0 1 2 3 4 5 6 7|8 9 0 1 2 3 4 5|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |fmt|     0     |   cs id - 64  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// Basic chunk header 3
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |0 1 2 3 4 5 6 7|8 9 0 1 2 3 4 5|6 7 8 9 0 1 2 3|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |fmt|     1     |        cs id - 64             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type Basic struct {
	FMT  byte
	CSID uint32
	Flag uint8
}

// Header of chunk message
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |0 1 2 3 4 5 6 7|8 9 0 1 2 3 4 5|6 7 8 9 0 1 2 3|4 5 6 7 8 9 0 1|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |               timestamp [delta]               |message length |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |     message length (cont)     |    type id    |   stream id   |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |               stream id (cont)                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// Header of message
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |0 1 2 3 4 5 6 7|8 9 0 1 2 3 4 5|6 7 8 9 0 1 2 3|4 5 6 7 8 9 0 1|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |    type id    |           payload length (3 bytes)            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                      timestamp (4 bytes)                      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |              stream id (3 bytes)              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type Header struct {
	TypeID    byte
	Length    uint32
	Timestamp uint32
	StreamID  uint32
}

// Message of RTMP
type Message struct {
	Basic
	Header
	Buffer  bytes.Buffer
	Payload []byte
}

// Parse tries to read a message from the given data
func (me *Message) Parse(data []byte) (int, error) {
	n := len(data)
	i := 0

	if n < 11 {
		return i, fmt.Errorf("data not enough: %d/11", n)
	}

	me.TypeID = data[i]
	i++

	me.Length = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
	i += 3

	me.Timestamp = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2]) | uint32(data[i+3])<<24
	i += 4

	me.StreamID = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
	i += 3

	if remains := n - i; remains < int(me.Length) {
		return i, fmt.Errorf("data not enough: %d/%d", remains, me.Length)
	}

	me.Payload = data[i : i+int(me.Length)]
	i += int(me.Length)

	return i, nil
}

// New creates a message
func New() *Message {
	return new(Message)
}
