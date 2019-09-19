package message

import (
	"encoding/binary"
	"fmt"
)

// BandwidthMessage of RTMP
type BandwidthMessage struct {
	Message
	AckWindowSize uint32
	LimitType     byte
}

// Parse tries to read a bandwidth message from the given data
func (me *BandwidthMessage) Parse(data []byte) (int, error) {
	n := len(data)
	i := 0

	if n < 5 {
		return i, fmt.Errorf("data not enough: %d/5", n)
	}

	me.AckWindowSize = binary.BigEndian.Uint32(data[i : i+4])
	i += 4

	me.LimitType = data[i]
	i++

	return i, nil
}
