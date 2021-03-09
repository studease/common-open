package message

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format/flv"
)

// VideoMessage of RTMP.
type VideoMessage struct {
	av.Packet
}

// Init this class.
func (me *VideoMessage) Init() *VideoMessage {
	me.Packet.Init()
	return me
}

// Parse tries to read a video message from the given data.
func (me *VideoMessage) Parse(data []byte) (int, error) {
	n := len(data)
	i := 0

	if n == 0 {
		return i, nil
	}

	if n < int(me.Length) {
		return i, fmt.Errorf("data not enough: %d/%d", n, me.Length)
	}

	b := data[i]

	switch b & 0x0F {
	case flv.AVC:
		me.Codec = "AVC"
	}

	me.Kind = av.KindVideo

	frametype := b >> 4
	me.Set("FrameType", frametype)
	me.Set("Keyframe", frametype == flv.KEYFRAME || frametype == flv.GENERATED_KEYFRAME)
	i++

	me.Position = uint32(i)
	me.Set("DataType", data[i])
	i++

	me.Payload = data
	return n, nil
}
