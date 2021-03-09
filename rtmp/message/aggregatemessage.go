package message

import (
	"container/list"
	"fmt"

	CSID "github.com/studease/common/rtmp/message/csid"
)

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
		sub := New()
		sub.FMT = 0
		sub.CSID = CSID.COMMAND_2
		sub.Flag = FLAG_ABSOLUTE

		j, err := sub.Parse(data[i:])
		if err != nil {
			return i, err
		}

		i += j + 4
		me.Subs.PushBack(sub)
	}

	return i, nil
}
