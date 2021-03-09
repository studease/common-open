package message

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format/flv"
)

// AudioMessage of RTMP.
type AudioMessage struct {
	av.Packet
}

// Init this class.
func (me *AudioMessage) Init() *AudioMessage {
	me.Packet.Init()
	return me
}

// Parse tries to read a audio message from the given data.
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
		me.Codec = "AAC"
	}

	me.Kind = av.KindAudio
	me.Set("SampleRate", (b>>2)&0x03)
	me.Set("SampleSize", (b>>1)&0x01)
	me.Set("SampleType", b&0x01)
	i++

	me.Position = uint32(i)
	me.Set("DataType", data[i])
	i++

	me.Payload = data
	return n, nil
}
