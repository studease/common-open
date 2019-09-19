package message

import (
	"encoding/binary"
	"fmt"

	EventType "github.com/studease/common/rtmp/message/eventtype"
)

// UserControlEvent of user control message
type UserControlEvent struct {
	Type         uint16
	StreamID     uint32
	BufferLength uint32
	Timestamp    uint32
}

// UserControlMessage of RTMP
type UserControlMessage struct {
	Message
	Event UserControlEvent
}

// Parse tries to read a user control message from the given data
func (me *UserControlMessage) Parse(data []byte) (int, error) {
	n := len(data)
	i := 0

	if n < int(me.Length) {
		return i, fmt.Errorf("data not enough: %d/%d", n, me.Length)
	}

	me.Event.Type = binary.BigEndian.Uint16(data[i : i+2])
	i += 2

	b := data[i:]

	switch me.Event.Type {
	case EventType.SET_BUFFER_LENGTH:
		me.Event.BufferLength = binary.BigEndian.Uint32(b[4:])
		i += 4
		fallthrough
	case EventType.STREAM_BEGIN:
		fallthrough
	case EventType.STREAM_EOF:
		fallthrough
	case EventType.STREAM_DRY:
		fallthrough
	case EventType.STREAM_IS_RECORDED:
		fallthrough
	case EventType.BUFFER_EMPTY:
		fallthrough
	case EventType.BUFFER_READY:
		me.Event.StreamID = binary.BigEndian.Uint32(b)
		i += 4

	case EventType.PING_REQUEST:
		fallthrough
	case EventType.PING_RESPONSE:
		me.Event.Timestamp = binary.BigEndian.Uint32(b)
		i += 4

	default:
		panic(fmt.Sprintf("unrecognized event type %02X", me.Event.Type))
	}

	return i, nil
}
