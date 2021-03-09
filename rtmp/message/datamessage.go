package message

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/utils/amf"
)

// DataMessage of RTMP.
type DataMessage struct {
	av.Packet
	Handler string
	Key     string
	Value   *amf.Value
}

// Init this class.
func (me *DataMessage) Init() *DataMessage {
	me.Packet.Init()
	return me
}

// Parse tries to read a data message from the given data.
func (me *DataMessage) Parse(data []byte) (int, error) {
	l := len(data)
	i := 0

	if l < int(me.Length) {
		return i, fmt.Errorf("data not enough: %d/%d", l, me.Length)
	}

	v := amf.NewValue(amf.STRING)

	n, err := amf.Decode(v, data[i:])
	if err != nil {
		return i, err
	}
	i += n

	me.Handler = v.String()
	me.Set("Handler", me.Handler)

	switch me.Handler {
	case "@setDataFrame":
		fallthrough
	case "@clearDataFrame":
		me.Payload = data[i:]

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}
		i += n

		me.Key = v.String()
		me.Set("Key", me.Key)

	default:
		me.Key = v.String()
		me.Set("Key", me.Key)
		me.Payload = data
	}

	n, err = amf.Decode(v, data[i:])
	if err != nil {
		return i, err
	}
	i += n

	me.Kind = av.KindScript
	me.Value = v
	me.Set("Value", me.Value)
	return i, nil
}
