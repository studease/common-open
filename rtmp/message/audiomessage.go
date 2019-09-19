package message

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/format/flv"
)

// AudioMessage of RTMP
type AudioMessage av.Packet

// Parse tries to read a audio message from the given data
func (me *AudioMessage) Parse(data []byte) (int, error) {
	n := len(data)
	i := 0

	if n == 0 {
		return i, nil
	}

	if n < int(me.Length) {
		return i, fmt.Errorf("data not enough: %d/%d", n, me.Length)
	}

	b := data[i]

	switch b & 0xF0 {
	case flv.AAC:
		me.Codec = codec.AAC
	}

	me.Type = av.TYPE_AUDIO
	me.SampleRate = (b >> 2) & 0x03
	me.SampleSize = (b >> 1) & 0x01
	me.SampleType = b & 0x01
	i++

	me.DataType = data[i]
	i++

	me.Payload = data

	return n, nil
}
