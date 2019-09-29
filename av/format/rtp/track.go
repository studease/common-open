package rtp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format"
	"github.com/studease/common/av/utils/sdp"
	"github.com/studease/common/log"
)

// Track inherit from MediaTrack, provides methods to format RTP packets
type Track struct {
	format.MediaTrack

	logger         log.ILogger
	transport      string
	parameters     map[string]func(string) error
	channel        byte
	control        byte
	sequenceNumber uint16
	psent          uint32
	osent          uint32
}

// Init this class
func (me *Track) Init(codec av.Codec, info *av.Information, logger log.ILogger, factory log.ILoggerFactory) *Track {
	me.MediaTrack.Init(codec, info, logger, factory)
	me.logger = logger
	me.parameters = map[string]func(string) error{
		"unicast":     me.processNothing,
		"multicast":   me.processMulticast,
		"destination": me.processNothing,
		"interleaved": me.processPort,
		"append":      me.processNothing,
		"ttl":         me.processNothing,
		"layers":      me.processNothing,
		"port":        me.processPort,
		"client_port": me.processPort,
		"server_port": me.processPort,
		"ssrc":        me.processNothing,
		"mode":        me.processNothing,
	}
	me.sequenceNumber = 0
	me.psent = 0
	me.osent = 0
	return me
}

// SetTransport sets the transport given as the argument
func (me *Track) SetTransport(transport string) error {
	arr := strings.Split(transport, ";")
	me.transport = arr[0]

	for i := 1; i < len(arr); i++ {
		item := arr[i]

		j := strings.IndexByte(item, '=')
		k := item
		v := ""

		if j != -1 {
			k = string([]byte(item)[:j])
			v = string([]byte(item)[j+1:])
		}

		if handler := me.parameters[k]; handler != nil {
			if err := handler(v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (me *Track) processMulticast(param string) error {
	return fmt.Errorf("%s not supported", param)
}

func (me *Track) processPort(param string) error {
	arr := strings.Split(param, "-")

	n, err := strconv.Atoi(arr[0])
	if err != nil {
		return err
	}

	me.channel = byte(n)

	n, err = strconv.Atoi(arr[1])
	if err != nil {
		return err
	}

	me.control = byte(n)

	return nil
}

func (me *Track) processNothing(param string) error {
	return nil
}

// Format returns a sequence of RTP packets with the given arguments
func (me *Track) Format(pkt *av.Packet) []*Packet {
	switch pkt.Codec {
	case codec.AAC:
		return me.getAACPackets(pkt)

	case codec.AVC:
		return me.getAVCPackets(pkt)

	default:
		panic(fmt.Sprintf("unrecognized codec 0x%02X", pkt.Codec))
	}
}

func (me *Track) getAACPackets(pkt *av.Packet) []*Packet {
	var (
		arr = make([]*Packet, 0)
	)

	ctx := me.Context.(*aac.Context)

	// rtptime/timestamp = rate/1000
	rtptime := int64(pkt.Timestamp) * int64(ctx.SamplingFrequency) / 1000

	size := MTU - 4
	if me.transport == sdp.RTP_AVP_TCP {
		// | 1 magic number | 1 channel number | 2 embedded data length |
		size -= 4
	}

	// | 12 RTP.Header |
	size -= 12

	n := len(ctx.Data)

	auHeader := make([]byte, 4)
	auHeader[0] = 0x00
	auHeader[1] = 0x10
	auHeader[2] = byte((n & 0x1FE0) >> 5)
	auHeader[3] = byte((n & 0x1F) << 3)

	count := n / size
	if count*size < n {
		count++
	}

	for i, x := 0, 0; x < count; x++ {
		if x == count-1 {
			size = n - i
		}

		me.sequenceNumber++

		pkt := new(Packet).Init()
		pkt.M = 1
		pkt.PT = 96
		pkt.SN = me.sequenceNumber
		pkt.Timestamp = uint32(rtptime)
		pkt.SSRC = uint32(me.ID())
		pkt.Payload = append(pkt.Payload, auHeader...)
		pkt.Payload = append(pkt.Payload, ctx.Data[i:i+size]...)

		i += size
		arr = append(arr, pkt)
	}

	return arr
}

func (me *Track) getAVCPackets(pkt *av.Packet) []*Packet {
	var (
		arr      = make([]*Packet, 0)
		S   byte = 0x80
		E   byte = 0x00
		R   byte = 0x00
	)

	ctx := me.Context.(*avc.Context)

	// rtptime/timestamp = rate/1000
	rtptime := int64(pkt.Timestamp) * H264_FREQ / 1000

	size := MTU
	if me.transport == sdp.RTP_AVP_TCP {
		// | 1 magic number | 1 channel number | 2 embedded data length |
		size -= 4
	}

	// | 12 RTP.Header | 1 F NRI Type |
	size -= 13

	for _, unit := range ctx.NALUs {
		ctx.ForbiddenZeroBit = unit[0] >> 7
		ctx.NalRefIdc = (unit[0] & 0x60) >> 5
		ctx.NalUnitType = unit[0] & 0x1F

		if ctx.NalUnitType == avc.NAL_SPS || ctx.NalUnitType == avc.NAL_PPS {
			continue
		}

		// Insert SPS PPS before keyframe
		if ctx.NalUnitType == avc.NAL_IDR_SLICE {
			me.sequenceNumber++

			x := len(ctx.SPS.Data)
			y := len(ctx.PPS.Data)

			pkt := new(Packet).Init()
			pkt.PT = 96
			pkt.SN = me.sequenceNumber
			pkt.Timestamp = uint32(rtptime)
			pkt.SSRC = uint32(me.ID())
			pkt.Payload = append(pkt.Payload, []byte{
				(ctx.SPS.Data[0] & 0x60) | NAL_STAP_A,
				byte(x >> 8), byte(x),
			}...)
			pkt.Payload = append(pkt.Payload, ctx.SPS.Data...)
			pkt.Payload = append(pkt.Payload, []byte{
				byte(y >> 8), byte(y),
			}...)
			pkt.Payload = append(pkt.Payload, ctx.PPS.Data...)

			arr = append(arr, pkt)
		}

		// Frame data
		if n := len(unit); n <= size { // Single NAL Unit Packet
			me.sequenceNumber++

			pkt := new(Packet).Init()
			pkt.PT = 96
			pkt.SN = me.sequenceNumber
			pkt.Timestamp = uint32(rtptime)
			pkt.SSRC = uint32(me.ID())
			pkt.Payload = make([]byte, n)
			copy(pkt.Payload, unit)

			arr = append(arr, pkt)
		} else { // FU-A
			// FU header
			size--

			count := n / size
			if count*size < n {
				count++
			}

			// Fragments
			for i, x := 1, 0; x < count; x++ {
				if x > 0 {
					S = 0x00
				}

				if x == count-1 {
					E = 0x40
					size = n - i
				}

				me.sequenceNumber++

				pkt := new(Packet).Init()
				pkt.PT = 96
				pkt.SN = me.sequenceNumber
				pkt.Timestamp = uint32(rtptime)
				pkt.SSRC = uint32(me.ID())
				pkt.Payload = make([]byte, 2+size)
				copy(pkt.Payload[:2], []byte{
					(unit[0] & 0x60) | NAL_FU_A,
					S | E | R | ctx.NalUnitType,
				})
				copy(pkt.Payload[2:], unit[i:i+size])

				i += size
				arr = append(arr, pkt)
			}
		}
	}

	return arr
}
