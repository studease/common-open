package rtcp

import (
	"bytes"
	"time"
)

// Static constants.
const (
	Version byte = 2
)

// FormatSR returns a formated SR packet with the given arguments.
func FormatSR(ssrc, timestamp, psent, osent uint32) []byte {
	var (
		b      bytes.Buffer
		length = 6
		now    = time.Now()
		t64    int64
		msw    uint32
		lsw    uint32
	)

	t64 = now.Unix() + 0x83AA7E80
	msw = uint32(t64)

	t64 = now.Unix() - now.UnixNano()
	lsw = uint32((t64 << 32) / 1000000000)

	b.Write([]byte{
		Version << 6,
		TYPE_SR,
		byte(length >> 8), byte(length),
		byte(ssrc >> 24), byte(ssrc >> 16), byte(ssrc >> 8), byte(ssrc),
		byte(msw >> 24), byte(msw >> 16), byte(msw >> 8), byte(msw),
		byte(lsw >> 24), byte(lsw >> 16), byte(lsw >> 8), byte(lsw),
		byte(timestamp >> 24), byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp),
		byte(psent >> 24), byte(psent >> 16), byte(psent >> 8), byte(psent),
		byte(osent >> 24), byte(osent >> 16), byte(osent >> 8), byte(osent),
	})

	return b.Bytes()
}

// FormatSDES returns a formated SDES packet with the given arguments.
func FormatSDES(ssrc uint32, items ...Item) []byte {
	var (
		b      bytes.Buffer
		length = 7
	)

	b.Write([]byte{
		(Version << 6) | 0x01,
		TYPE_SDES,
		byte(length >> 8), byte(length),
		byte(ssrc >> 24), byte(ssrc >> 16), byte(ssrc >> 8), byte(ssrc),
	})
	for _, item := range items {
		b.Write([]byte{
			item.Type,
			item.Length,
		})
		b.Write(item.Data)
	}

	return b.Bytes()
}
