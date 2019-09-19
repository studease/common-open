package message

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/format/flv"
)

// VideoMessage of RTMP
type VideoMessage av.Packet

// Parse tries to read a video message from the given data
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
		me.Codec = codec.AVC
	}

	me.Type = av.TYPE_VIDEO
	me.FrameType = b >> 4
	i++

	me.DataType = data[i]
	i++

	me.Payload = data

	return n, nil
}
