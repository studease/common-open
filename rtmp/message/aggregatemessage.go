package message

import (
	"container/list"
	"encoding/binary"
	"fmt"
)

// SubMessage of aggregate message
type SubMessage struct {
	Message
	BackPointer uint32
}

// Parse tries to read a sub message from the given data
func (me *SubMessage) Parse(data []byte) (int, error) {
	n := len(data)

	i, err := me.Message.Parse(data)
	if err != nil {
		return i, err
	}

	if remains := n - i; remains < 4 {
		return i, fmt.Errorf("data not enough: %d/%d", remains, 4)
	}

	me.BackPointer = binary.BigEndian.Uint32(data[i : i+4])
	i += 4

	return i, nil
}

// AggregateMessage of RTMP
type AggregateMessage struct {
	Message
	Subs list.List
}

// Parse tries to read a aggregate message from the given data
func (me *AggregateMessage) Parse(data []byte) (int, error) {
	n := len(data)
	i := 0

	if n < int(me.Length) {
		return 0, fmt.Errorf("data not enough: %d/%d", n, me.Length)
	}

	for i < n {
		sub := new(SubMessage)

		j, err := sub.Parse(data[i:])
		if err != nil {
			return i, err
		}

		i += j
		me.Subs.PushBack(sub)
	}

	return i, nil
}
