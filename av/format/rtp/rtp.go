package rtp

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format"
	"github.com/studease/common/av/utils/sdp"
	"github.com/studease/common/events"
	ErrorEvent "github.com/studease/common/events/errorevent"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	MediaStreamTrackEvent "github.com/studease/common/events/mediastreamtrackevent"
	"github.com/studease/common/log"
)

// Static constants.
const (
	Version   byte  = 2
	MTU       int   = 1500
	H264_FREQ int64 = 90000
)

// NAL types.
const (
	NAL_UNIT   = 23
	NAL_STAP_A = 24
	NAL_STAP_B = 25
	NAL_MTAP16 = 26
	NAL_MTAP24 = 27
	NAL_FU_A   = 28
	NAL_FU_B   = 29
)

func init() {
	format.Register("RTP", RTP{})
}

// RTP MediaStream, implements IDemuxer, IRemuxer.
type RTP struct {
	format.MediaStream

	Mode       uint32
	logger     log.ILogger
	mtx        sync.RWMutex
	source     av.IMediaStream
	handlers   map[string]func(track av.IMediaStreamTrack, pkt *av.Packet)
	readyState uint32

	addtrackListener    *events.EventListener
	removetrackListener *events.EventListener
	packetListener      *events.EventListener
	errorListener       *events.EventListener
	closeListener       *events.EventListener
}

// Init this class.
func (me *RTP) Init(mode uint32, logger log.ILogger) av.IRemuxer {
	me.MediaStream.Init(logger)
	me.Mode = mode
	me.logger = logger
	me.readyState = format.RemuxInactive
	me.addtrackListener = events.NewListener(me.onAddTrack, 0)
	me.removetrackListener = events.NewListener(me.onRemoveTrack, 0)
	me.packetListener = events.NewListener(me.onPacket, 0)
	me.errorListener = events.NewListener(me.onError, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	me.handlers = map[string]func(track av.IMediaStreamTrack, pkt *av.Packet){
		"AVC": me.getAVCPackets,
		"AAC": me.getAACPackets,
	}
	return me
}

// Append parses buffer.
func (me *RTP) Append(data []byte) {
	// TODO(spencerlau): Parse fmp4 boxes
	me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "NotSupportedError", fmt.Errorf("The operation is not supported")))
}

// Reset clears IDemuxer cache, and closes IMediaStream.
func (me *RTP) Reset() {
	me.MediaStream.Close()
	me.Init(me.Mode, me.logger)
}

// Source attaches the IMediaStream as input.
func (me *RTP) Source(ms av.IMediaStream) {
	if ms == nil {
		me.Close()
		return
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	me.source = ms
	atomic.StoreUint32(&me.readyState, format.RemuxWaiting)

	onMetaData := ms.GetDataFrame("onMetaData")
	if onMetaData != nil {
		me.SetDataFrame("onMetaData", onMetaData)
	}
	tracks := ms.GetTracks()
	for _, item := range tracks {
		if item.Kind() == format.KindVideo && (me.Mode&av.ModeVideo&av.ModeKeyframe) == 0 || item.Kind() == format.KindAudio && (me.Mode&av.ModeAudio) == 0 {
			continue
		}
		track := item.Clone()
		me.AddTrack(track)
		source := track.Source()
		if infoframe := source.GetInfoFrame(); (me.Mode&av.ModeInterleaved) == 0 && infoframe != nil {
			me.GetInitSegment(track)
		}
		source.AddEventListener(MediaEvent.PACKET, me.packetListener)
	}

	ms.AddEventListener(MediaStreamTrackEvent.ADDTRACK, me.addtrackListener)
	ms.AddEventListener(MediaStreamTrackEvent.REMOVETRACK, me.removetrackListener)
	ms.AddEventListener(MediaEvent.PACKET, me.packetListener)
	ms.AddEventListener(ErrorEvent.ERROR, me.errorListener)
	ms.AddEventListener(Event.CLOSE, me.closeListener)
}

func (me *RTP) onAddTrack(e *MediaStreamTrackEvent.MediaStreamTrackEvent) {
	switch e.Track.Kind() {
	case format.KindVideo:
		if (me.Mode & av.ModeVideo & av.ModeKeyframe) == 0 {
			return
		}
	case format.KindAudio:
		if (me.Mode & av.ModeAudio) == 0 {
			return
		}
	default:
		me.logger.Debugf(2, "Ignored unrecognized track: kind=%s.", e.Track.Kind())
		return
	}

	source := e.Track.Source()
	if me.Attached(source) == nil {
		track := new(MediaStreamTrack).Init(e.Track.Kind(), source, me.logger)
		me.AddTrack(track)
		source.AddEventListener(MediaEvent.PACKET, me.packetListener)
	}
}

func (me *RTP) onRemoveTrack(e *MediaStreamTrackEvent.MediaStreamTrackEvent) {
	source := e.Track.Source()
	track := me.Attached(source)
	if track != nil {
		source.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
		me.RemoveTrack(track)
	}
}

func (me *RTP) onPacket(e *MediaEvent.MediaEvent) {
	switch e.Packet.Kind {
	case av.KindAudio:
		me.onAudioPacket(e.Packet)
	case av.KindVideo:
		me.onVideoPacket(e.Packet)
	case av.KindScript:
		me.onDataPacket(e.Packet)
	default:
		me.logger.Errorf("Unrecognized packet: %s", e.Packet.Kind)
	}
}

func (me *RTP) onDataPacket(pkt *av.Packet) {
	key := pkt.Get("Key").(string)
	me.SetDataFrame(key, pkt)
}

func (me *RTP) onAudioPacket(pkt *av.Packet) {
	track := me.GetAudioTracks()[0]
	source := track.Source()

	switch pkt.Codec {
	case "AAC":
		switch pkt.Get("DataType").(byte) {
		case aac.SPECIFIC_CONFIG:
			if (me.Mode & av.ModeInterleaved) == 0 {
				me.GetInitSegment(track)
			}
		case aac.RAW_FRAME_DATA:
			if source.GetInfoFrame() == nil || atomic.LoadUint32(&me.readyState) != format.RemuxPumping {
				return
			}
			me.GetSegment(track, pkt)
		default:
			me.logger.Errorf("Unrecognized AAC packet type: 0x%02X", pkt.Get("DataType").(byte))
		}
	default:
		me.logger.Errorf("Unrecognized codec: %s", pkt.Codec)
	}
}

func (me *RTP) onVideoPacket(pkt *av.Packet) {
	track := me.GetVideoTracks()[0]
	source := track.Source()

	switch pkt.Codec {
	case "AVC":
		switch pkt.Get("DataType").(byte) {
		case avc.SEQUENCE_HEADER:
			if (me.Mode & av.ModeInterleaved) == 0 {
				me.GetInitSegment(track)
			}
		case avc.NALU:
			if pkt.Get("Keyframe").(bool) && atomic.CompareAndSwapUint32(&me.readyState, format.RemuxWaiting, format.RemuxPumping) {
				me.Info.TimeBase = pkt.Timestamp
				// If you are willing to use interleaved mode, all of the info frames should be ahead of any media frames.
				audios := me.GetAudioTracks()
				if (me.Mode&av.ModeAudio) == 0 || len(audios) == 0 || (me.Mode&av.ModeInterleaved) != 0 && audios[0].Source().GetInfoFrame() != nil {
					tracks := me.GetTracks()
					me.GetInitSegment(tracks...)
				}
			}
			if source.GetInfoFrame() == nil || atomic.LoadUint32(&me.readyState) != format.RemuxPumping || (me.Mode&av.ModeKeyframe) == av.ModeKeyframe && !pkt.Get("Keyframe").(bool) {
				return
			}
			fallthrough
		case avc.END_OF_SEQUENCE:
			me.GetSegment(track, pkt)
		default:
			me.logger.Errorf("Unrecognized AVC packet type: 0x%02X", pkt.Get("DataType").(byte))
		}
	default:
		me.logger.Errorf("Unrecognized codec: %s", pkt.Codec)
	}
}

func (me *RTP) onError(e *ErrorEvent.ErrorEvent) {
	me.logger.Debugf(0, "%s: %s", e.Name, e.Message)
	me.Close()
}

func (me *RTP) onClose(e *Event.Event) {
	me.Close()
}

// GetInitSegment does nothing while the source handler needs to deal with the info frame itself.
func (me *RTP) GetInitSegment(tracks ...av.IMediaStreamTrack) {
	// Do nothing here.
}

// GetSegment generates a sequence of RTP packets of the given track with the packet.
func (me *RTP) GetSegment(track av.IMediaStreamTrack, pkt *av.Packet) {
	if h := me.handlers[pkt.Codec]; h != nil {
		h(track, pkt)
	}
}

func (me *RTP) getAVCPackets(track av.IMediaStreamTrack, pkt *av.Packet) {
	var (
		S byte = 0x80
		E byte = 0x00
		R byte = 0x00
	)

	trak := track.(*MediaStreamTrack)
	source := track.Source().(*avc.AVC)

	// rtptime/timestamp = rate/1000
	rtptime := int64(pkt.Timestamp) * H264_FREQ / 1000

	size := MTU
	if trak.Transport == sdp.RTP_AVP_TCP {
		// | 1 magic number | 1 channel number | 2 embedded data length |
		size -= 4
	}
	// | 12 RTP.Header | 1 F NRI Type |
	size -= 13

	nalUnitType := pkt.Get("NalUnitType").(int)
	nalus := pkt.Get("NALUs").([][]byte)
	for _, unit := range nalus {
		if nalUnitType == avc.NAL_SPS || nalUnitType == avc.NAL_PPS {
			continue
		}

		// Insert SPS PPS before keyframe
		if nalUnitType == avc.NAL_IDR_SLICE {
			trak.SN++
			if trak.SN > 0xFFFF {
				trak.SN >>= 16
			}

			i := 0
			x := len(source.SPS.Data)
			y := len(source.PPS.Data)

			dst := new(av.Packet).Init()
			dst.Kind = pkt.Kind
			dst.Codec = pkt.Codec
			dst.Length = uint32(5 + x + y)
			dst.Timestamp = pkt.Timestamp
			dst.StreamID = uint32(track.ID())
			dst.Set("V", Version)          // 2 bits
			dst.Set("P", byte(0))          // 1 bit
			dst.Set("X", byte(0))          // 1 bit
			dst.Set("CC", byte(0))         // 4 bits
			dst.Set("M", byte(0))          // 1 bit
			dst.Set("PT", byte(96))        // 7 bits
			dst.Set("SN", uint16(trak.SN)) // 2 bytes
			dst.Set("Timestamp", uint32(rtptime))
			dst.Set("SSRC", dst.StreamID)
			dst.Set("CSRC", []uint32{})
			dst.Payload = make([]byte, dst.Length)
			copy(dst.Payload[i:], []byte{
				(source.SPS.Data[0] & 0x60) | NAL_STAP_A,
				byte(x >> 8), byte(x),
			})
			i += 3
			copy(dst.Payload[i:], source.SPS.Data)
			i += x
			copy(dst.Payload[i:], []byte{
				byte(y >> 8), byte(y),
			})
			i += 2
			copy(dst.Payload[i:], source.PPS.Data)
			i += y
			me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, dst))
		}

		// Frame data
		if n := len(unit); n <= size { // Single NAL Unit Packet
			trak.SN++
			if trak.SN > 0xFFFF {
				trak.SN >>= 16
			}

			dst := new(av.Packet).Init()
			dst.Kind = pkt.Kind
			dst.Codec = pkt.Codec
			dst.Length = uint32(n)
			dst.Timestamp = pkt.Timestamp
			dst.StreamID = uint32(track.ID())
			dst.Set("V", Version)          // 2 bits
			dst.Set("P", byte(0))          // 1 bit
			dst.Set("X", byte(0))          // 1 bit
			dst.Set("CC", byte(0))         // 4 bits
			dst.Set("M", byte(0))          // 1 bit
			dst.Set("PT", byte(96))        // 7 bits
			dst.Set("SN", uint16(trak.SN)) // 2 bytes
			dst.Set("Timestamp", uint32(rtptime))
			dst.Set("SSRC", dst.StreamID)
			dst.Set("CSRC", []uint32{})
			dst.Payload = make([]byte, dst.Length)
			copy(dst.Payload, unit)
			me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, dst))
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

				trak.SN++
				if trak.SN > 0xFFFF {
					trak.SN >>= 16
				}

				dst := new(av.Packet).Init()
				dst.Kind = pkt.Kind
				dst.Codec = pkt.Codec
				dst.Length = uint32(2 + size)
				dst.Timestamp = pkt.Timestamp
				dst.StreamID = uint32(track.ID())
				dst.Set("V", Version)          // 2 bits
				dst.Set("P", byte(0))          // 1 bit
				dst.Set("X", byte(0))          // 1 bit
				dst.Set("CC", byte(0))         // 4 bits
				dst.Set("M", byte(0))          // 1 bit
				dst.Set("PT", byte(96))        // 7 bits
				dst.Set("SN", uint16(trak.SN)) // 2 bytes
				dst.Set("Timestamp", uint32(rtptime))
				dst.Set("SSRC", dst.StreamID)
				dst.Set("CSRC", []uint32{})
				dst.Payload = make([]byte, dst.Length)
				copy(dst.Payload[:2], []byte{
					(unit[0] & 0x60) | NAL_FU_A,
					S | E | R | byte(nalUnitType),
				})
				copy(dst.Payload[2:], unit[i:i+size])
				i += size
				me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, dst))
			}
		}
	}
}

func (me *RTP) getAACPackets(track av.IMediaStreamTrack, pkt *av.Packet) {
	trak := track.(*MediaStreamTrack)
	source := track.Source().(*aac.AAC)

	// rtptime/timestamp = rate/1000
	rtptime := int64(pkt.Timestamp) * int64(source.SamplingFrequency) / 1000

	size := MTU - 4
	if trak.Transport == sdp.RTP_AVP_TCP {
		// | 1 magic number | 1 channel number | 2 embedded data length |
		size -= 4
	}
	// | 12 RTP.Header |
	size -= 12

	data := pkt.Get("Data").([]byte)
	n := len(data)

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

		trak.SN++
		if trak.SN > 0xFFFF {
			trak.SN >>= 16
		}

		dst := new(av.Packet).Init()
		dst.Kind = pkt.Kind
		dst.Codec = pkt.Codec
		dst.Length = uint32(4 + size)
		dst.Timestamp = pkt.Timestamp
		dst.StreamID = uint32(track.ID())
		dst.Set("V", Version)          // 2 bits
		dst.Set("P", byte(0))          // 1 bit
		dst.Set("X", byte(0))          // 1 bit
		dst.Set("CC", byte(0))         // 4 bits
		dst.Set("M", byte(1))          // 1 bit
		dst.Set("PT", byte(96))        // 7 bits
		dst.Set("SN", uint16(trak.SN)) // 2 bytes
		dst.Set("Timestamp", uint32(rtptime))
		dst.Set("SSRC", dst.StreamID)
		dst.Set("CSRC", []uint32{})
		dst.Payload = make([]byte, dst.Length)
		copy(dst.Payload[:4], auHeader)
		copy(dst.Payload[4:], data[i:i+size])
		i += size
		me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, dst))
	}
}

// Format returns the raw data of packet, formated in RTP packet.
func Format(pkt *av.Packet) []byte {
	v := pkt.Get("V").(byte)
	p := pkt.Get("P").(byte)
	x := pkt.Get("X").(byte)
	cc := pkt.Get("CC").(byte)
	m := pkt.Get("M").(byte)
	pt := pkt.Get("PT").(byte)
	sn := pkt.Get("SN").(uint16)
	timestamp := pkt.Get("Timestamp").(uint32)
	ssrc := pkt.Get("SSRC").(uint32)
	csrc := pkt.Get("CSRC").([]uint32)
	i := 0
	n := 12 + len(csrc)*4 + len(pkt.Payload)

	dst := make([]byte, n)
	copy(dst[:12], []byte{
		v<<6 | p<<5 | x<<4 | cc,
		m<<7 | pt,
		byte(sn >> 8), byte(sn),
		byte(timestamp >> 24), byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp),
		byte(ssrc >> 24), byte(ssrc >> 16), byte(ssrc >> 8), byte(ssrc),
	})
	i += 12
	for _, item := range csrc {
		copy(dst[i:], []byte{
			byte(item >> 24), byte(item >> 16), byte(item >> 8), byte(item),
		})
		i += 4
	}
	return dst
}

// SetDataFrame stores a data frame with the given key.
func (me *RTP) SetDataFrame(key string, pkt *av.Packet) {
	me.Info = *me.source.Information()
	me.MediaStream.SetDataFrame(key, pkt)
}

// Close detaches IRemuxer source, and closes IMediaStream.
func (me *RTP) Close() {
	switch atomic.LoadUint32(&me.readyState) {
	case format.RemuxWaiting:
		fallthrough
	case format.RemuxPumping:
		me.mtx.Lock()
		defer me.mtx.Unlock()

		atomic.StoreUint32(&me.readyState, format.RemuxInactive)
		me.DispatchEvent(Event.New(Event.CLOSE, me))

		tracks := me.GetTracks()
		for _, item := range tracks {
			source := item.Source()
			source.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
		}
		if ms := me.source; ms != nil {
			ms.RemoveEventListener(MediaStreamTrackEvent.ADDTRACK, me.addtrackListener)
			ms.RemoveEventListener(MediaStreamTrackEvent.REMOVETRACK, me.removetrackListener)
			ms.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
			ms.RemoveEventListener(ErrorEvent.ERROR, me.errorListener)
			ms.RemoveEventListener(Event.CLOSE, me.closeListener)
			me.source = nil
		}
		me.MediaStream.Close()
	}
}
