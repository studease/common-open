package message

import (
	"fmt"

	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/rtmp/message/command"
)

// CommandMessage of RTMP
type CommandMessage struct {
	Message
	CommandName    string
	TransactionID  uint64
	CommandObject  amf.Value
	Arguments      amf.Value
	StreamName     string
	Start          float64
	Duration       float64
	Reset          bool
	Flag           bool
	PublishingName string
	PublishingType string
	MilliSeconds   float64
}

// Parse tries to read a command message from the given data
func (me *CommandMessage) Parse(data []byte) (int, error) {
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
	me.CommandName = v.String()

	n, err = amf.Decode(v, data[i:])
	if err != nil {
		return i, err
	}

	i += n
	me.TransactionID = uint64(v.Double())

	switch me.CommandName {
	// NetConnection Commands
	case command.CONNECT:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n

		n, err = amf.Decode(&me.Arguments, data[i:])
		if err != nil {
			return i, err
		}

		i += n

	case command.CLOSE:
		// Do nothing here.

	case command.CREATE_STREAM:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n

	case command.RESULT:
		fallthrough
	case command.ERROR:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n

		n, err = amf.Decode(&me.Arguments, data[i:])
		if err != nil {
			return i, err
		}

		i += n

	// NetStream Commands
	case command.PLAY:
		me.Start = -2
		me.Duration = -1
		me.Reset = true

		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.StreamName = v.String()

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, nil
		}

		i += n
		me.Start = v.Double()

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, nil
		}

		i += n
		me.Duration = v.Double()

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, nil
		}

		i += n
		me.Reset = v.Bool()

	case command.PLAY2:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(&me.Arguments, data[i:])
		if err != nil {
			return i, err
		}

		i += n

	case command.DELETE_STREAM:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(&me.Arguments, data[i:])
		if err != nil {
			return i, err
		}

		i += n

	case command.CLOSE_STREAM:
		// Do nothing here.

	case command.RECEIVE_AUDIO:
		fallthrough
	case command.RECEIVE_VIDEO:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(&me.Arguments, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.Flag = v.Bool()

	case command.PUBLISH:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.PublishingName = v.String()

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.PublishingType = v.String()

	case command.SEEK:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.MilliSeconds = v.Double()

	case command.PAUSE:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.Flag = v.Bool()

		n, err = amf.Decode(v, data[i:])
		if err != nil {
			return i, err
		}

		i += n
		me.MilliSeconds = v.Double()

	case command.ON_STATUS:
		n, err = amf.Decode(&me.CommandObject, data[i:])
		if err != nil {
			return i, err
		}

		i += n // Type == amf.NULL

		n, err = amf.Decode(&me.Arguments, data[i:])
		if err != nil {
			return i, err
		}

		i += n

	default:
		// User command? Keep going...
	}

	return i, nil
}
