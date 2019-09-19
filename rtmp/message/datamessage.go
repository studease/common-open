package message

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/utils/amf"
)

// DataMessage of RTMP
type DataMessage av.Packet

// Parse tries to read a data message from the given data
func (me *DataMessage) Parse(data []byte) (int, error) {
	l := len(data)
	i := 0

	if l < int(me.Length) {
		return i, fmt.Errorf("data not enough: %d/%d", l, me.Length)
	}

	v := amf.NewValue()

	n, err := amf.Decode(v, data[i:])
	if err != nil {
		return i, err
	}

	i += n
	me.Handler = v.String()

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

	default:
		me.Key = me.Handler
		me.Payload = data
	}

	n, err = amf.Decode(v, data[i:])
	if err != nil {
		return i, err
	}

	i += n
	me.Type = av.TYPE_DATA
	me.Value = v

	return i, nil
}
