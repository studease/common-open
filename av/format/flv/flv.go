package flv

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/events"
	ErrorEvent "github.com/studease/common/events/errorevent"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	MediaStreamTrackEvent "github.com/studease/common/events/mediastreamtrackevent"
	"github.com/studease/common/log"
	Math "github.com/studease/common/utils/math"
)

// Tag kinds.
const (
	KindAudio  byte = 0x08
	KindVideo  byte = 0x09
	KindScript byte = 0x12
)

// RTMP/FLV video frame types.
const (
	KEYFRAME               = 0x1
	INTER_FRAME            = 0x2
	DISPOSABLE_INTER_FRAME = 0x3
	GENERATED_KEYFRAME     = 0x4
	INFO_OR_COMMAND_FRAME  = 0x5
)

// Video codecs.
const (
	JPEG           byte = 0x01
	H263           byte = 0x02
	SCREEN_VIDEO   byte = 0x03
	VP6            byte = 0x04
	VP6_ALPHA      byte = 0x05
	SCREEN_VIDEO_2 byte = 0x06
	AVC            byte = 0x07
)

// Audio codecs.
const (
	LINEAR_PCM_PLATFORM_ENDIAN   byte = 0x00
	ADPCM                        byte = 0x10
	MP3                          byte = 0x20
	LINEAR_PCM_LITTLE_ENDIAN     byte = 0x30
	NELLYMOSER_16_kHz_MONO       byte = 0x40
	NELLYMOSER_8_kHz_MONO        byte = 0x50
	NELLYMOSER                   byte = 0x60
	G_711_A_LAW_LOGARITHMIC_PCM  byte = 0x70
	G_711_MU_LAW_LOGARITHMIC_PCM byte = 0x80
	RESERVED                     byte = 0x90
	AAC                          byte = 0xA0
	SPEEX                        byte = 0xB0
	MP3_8_kHz                    byte = 0xE0
	DEVICE_SPECIFIC_SOUND        byte = 0xF0
)

// Static variables, should not change.
var (
	Rates  = []uint32{5500, 11025, 22050, 44100}
	Footer = []byte{
		0x17, 0x02, 0x00, 0x00, 0x00,
	}
)

// Static contants.
const (
	sw_f            = 0
	sw_l            = 1
	sw_v            = 2
	sw_version      = 3
	sw_flags        = 4
	sw_header0      = 5
	sw_header1      = 6
	sw_header2      = 7
	sw_header3      = 8
	sw_backpointer0 = 9
	sw_backpointer1 = 10
	sw_backpointer2 = 11
	sw_backpointer3 = 12
	sw_type         = 13
	sw_length0      = 14
	sw_length1      = 15
	sw_length2      = 16
	sw_timestamp0   = 17
	sw_timestamp1   = 18
	sw_timestamp2   = 19
	sw_timestamp3   = 20
	sw_streamid0    = 21
	sw_streamid1    = 22
	sw_streamid2    = 23
	sw_payload      = 24
)

// Header of FLV stream.
func Header(mode uint32) []byte {
	var (
		flags byte
	)

	if (mode & av.ModeVideo) != 0 {
		flags |= 0x01
	}
	if (mode & av.ModeAudio) != 0 {
		flags |= 0x04
	}
	return []byte{
		'F', 'L', 'V',
		0x01,
		flags,
		0x00, 0x00, 0x00, 0x09,
		0x00, 0x00, 0x00, 0x00,
	}
}

func init() {
	format.Register("FLV", FLV{})
}

// FLV MediaStream, implements IDemuxer, IRemuxer.
type FLV struct {
	format.MediaStream

	Mode        uint32
	logger      log.ILogger
	mtx         sync.RWMutex
	state       int
	hasAudio    bool
	hasVideo    bool
	backpointer uint32
	packet      *av.Packet
	source      av.IMediaStream
	readyState  uint32

	addtrackListener    *events.EventListener
	removetrackListener *events.EventListener
	packetListener      *events.EventListener
	errorListener       *events.EventListener
	closeListener       *events.EventListener
}

// Init this class.
func (me *FLV) Init(mode uint32, logger log.ILogger) av.IRemuxer {
	me.MediaStream.Init(logger)
	me.Mode = mode
	me.logger = logger
	me.state = sw_f
	me.hasAudio = false
	me.hasVideo = false
	me.backpointer = 0
	me.packet = nil
	me.readyState = format.RemuxInactive
	me.addtrackListener = events.NewListener(me.onAddTrack, 0)
	me.removetrackListener = events.NewListener(me.onRemoveTrack, 0)
	me.packetListener = events.NewListener(me.onPacket, 0)
	me.errorListener = events.NewListener(me.onError, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// Append parses buffer.
func (me *FLV) Append(data []byte) {
	size := len(data)
	for i := 0; i < size; i++ {
		switch me.state {
		case sw_f:
			if data[i] != 0x46 {
				me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "DataError", fmt.Errorf("Not \"F\"")))
				return
			}
			me.state = sw_l

		case sw_l:
			if data[i] != 0x4C {
				me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "DataError", fmt.Errorf("Not \"L\"")))
				return
			}
			me.state = sw_v

		case sw_v:
			if data[i] != 0x56 {
				me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "DataError", fmt.Errorf("Not \"V\"")))
				return
			}
			me.state = sw_version

		case sw_version:
			if data[i] != 0x01 {
				// Not strict
			}
			me.state = sw_flags

		case sw_flags:
			me.hasAudio = (data[i] & 0x04) == 0x04
			me.hasVideo = (data[i] & 0x01) == 0x01
			if !me.hasAudio && !me.hasVideo {
				// Not strict
			}
			me.state = sw_header0

		case sw_header0:
			me.state = sw_header1

		case sw_header1:
			me.state = sw_header2

		case sw_header2:
			me.state = sw_header3

		case sw_header3:
			me.state = sw_backpointer0

		case sw_backpointer0:
			me.backpointer = uint32(data[i]) << 24
			me.state = sw_backpointer1

		case sw_backpointer1:
			me.backpointer |= uint32(data[i]) << 16
			me.state = sw_backpointer2

		case sw_backpointer2:
			me.backpointer |= uint32(data[i]) << 8
			me.state = sw_backpointer3

		case sw_backpointer3:
			me.backpointer |= uint32(data[i])
			me.state = sw_type

		case sw_type:
			me.packet = new(av.Packet).Init()
			switch data[i] {
			case KindAudio:
				me.packet.Kind = av.KindAudio
			case KindVideo:
				me.packet.Kind = av.KindVideo
			case KindScript:
				me.packet.Kind = av.KindScript
			default:
				me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "TypeError", fmt.Errorf("Unrecognized flv tag 0x02X", data[i])))
				return
			}
			me.state = sw_length0

		case sw_length0:
			me.packet.Length = uint32(data[i]) << 16
			me.state = sw_length1

		case sw_length1:
			me.packet.Length |= uint32(data[i]) << 8
			me.state = sw_length2

		case sw_length2:
			me.packet.Length |= uint32(data[i])
			me.packet.Payload = make([]byte, me.packet.Length)
			me.packet.Position = 0
			me.state = sw_timestamp0

		case sw_timestamp0:
			me.packet.Timestamp = uint32(data[i]) << 16
			me.state = sw_timestamp1

		case sw_timestamp1:
			me.packet.Timestamp |= uint32(data[i]) << 8
			me.state = sw_timestamp2

		case sw_timestamp2:
			me.packet.Timestamp |= uint32(data[i])
			me.state = sw_timestamp3

		case sw_timestamp3:
			me.packet.Timestamp |= uint32(data[i]) << 24
			me.state = sw_streamid0

		case sw_streamid0:
			me.packet.StreamID = uint32(data[i]) << 16
			me.state = sw_streamid1

		case sw_streamid1:
			me.packet.StreamID |= uint32(data[i]) << 8
			me.state = sw_streamid2

		case sw_streamid2:
			me.packet.StreamID |= uint32(data[i])
			me.state = sw_payload

		case sw_payload:
			var n = Math.MinUint32(me.packet.Length-me.packet.Position, uint32(size-i))
			copy(me.packet.Payload[me.packet.Position:], data[i:i+int(n)])
			me.packet.Position += n
			i += int(n) - 1

			if me.packet.Position == me.packet.Length {
				switch me.packet.Kind {
				case av.KindAudio:
					me.packet.Set("Format", (me.packet.Payload[0]>>4)&0x0F)
					me.packet.Set("SampleRate", (me.packet.Payload[0]>>2)&0x03)
					me.packet.Set("SampleSize", (me.packet.Payload[0]>>1)&0x01)
					me.packet.Set("SampleType", me.packet.Payload[0]&0x01)
					me.packet.Set("DataType", me.packet.Payload[1]) // Extra parsing
					me.packet.Position = 1

				case av.KindVideo:
					frametype := (me.packet.Payload[0] >> 4) & 0x0F
					me.packet.Set("FrameType", frametype)
					me.packet.Set("Codec", me.packet.Payload[0]&0x0F)
					me.packet.Set("DataType", me.packet.Payload[1]) // Extra parsing
					me.packet.Set("Keyframe", frametype == KEYFRAME || frametype == GENERATED_KEYFRAME)
					me.packet.Position = 1

				case av.KindScript:
					me.packet.Position = 0
				}
				me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, me.packet))
				me.state = sw_backpointer0
			}

		default:
			me.DispatchEvent(ErrorEvent.New(ErrorEvent.ERROR, me, "InvalidStateError", fmt.Errorf("Invalid state while parsing flv tag")))
			return
		}
	}
}

// Reset clears IDemuxer cache, and closes IMediaStream.
func (me *FLV) Reset() {
	me.MediaStream.Close()
	me.Init(me.Mode, me.logger)
}

// Source attaches the IMediaStream as input.
func (me *FLV) Source(ms av.IMediaStream) {
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
		tag := me.format(onMetaData)
		if tag != nil {
			me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, tag))
		}
	}
	tracks := ms.GetTracks()
	for _, item := range tracks {
		if item.Kind() == format.KindVideo && (me.Mode&av.ModeVideo&av.ModeKeyframe) == 0 || item.Kind() == format.KindAudio && (me.Mode&av.ModeAudio) == 0 {
			continue
		}
		track := item.Clone()
		me.AddTrack(track)
		source := track.Source()
		if infoframe := source.GetInfoFrame(); infoframe != nil {
			tag := me.format(infoframe)
			if tag != nil {
				me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, tag))
			}
		}
		source.AddEventListener(MediaEvent.PACKET, me.packetListener)
	}

	ms.AddEventListener(MediaStreamTrackEvent.ADDTRACK, me.addtrackListener)
	ms.AddEventListener(MediaStreamTrackEvent.REMOVETRACK, me.removetrackListener)
	ms.AddEventListener(MediaEvent.PACKET, me.packetListener)
	ms.AddEventListener(ErrorEvent.ERROR, me.errorListener)
	ms.AddEventListener(Event.CLOSE, me.closeListener)
}

func (me *FLV) onAddTrack(e *MediaStreamTrackEvent.MediaStreamTrackEvent) {
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
		me.AddTrack(e.Track.Clone())
		source.AddEventListener(MediaEvent.PACKET, me.packetListener)
	}
}

func (me *FLV) onRemoveTrack(e *MediaStreamTrackEvent.MediaStreamTrackEvent) {
	source := e.Track.Source()
	track := me.Attached(source)
	if track != nil {
		source.RemoveEventListener(MediaEvent.PACKET, me.packetListener)
		me.RemoveTrack(track)
	}
}

func (me *FLV) onPacket(e *MediaEvent.MediaEvent) {
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

func (me *FLV) onDataPacket(pkt *av.Packet) {
	key := pkt.Get("Key").(string)
	me.SetDataFrame(key, pkt)

	switch key {
	case "onMetaData":
	default:
		me.logger.Debugf(2, "Ignored data frame: key=%s.", key)
		return
	}

	tag := me.format(pkt)
	if tag != nil {
		me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, tag))
	}
}

func (me *FLV) onAudioPacket(pkt *av.Packet) {
	track := me.GetAudioTracks()[0]
	source := track.Source()

	switch pkt.Codec {
	case "AAC":
		if source.GetInfoFrame() == nil || atomic.LoadUint32(&me.readyState) != format.RemuxPumping {
			return
		}
	default:
		me.logger.Errorf("Unrecognized codec: %s", pkt.Codec)
		return
	}

	tag := me.format(pkt)
	if tag != nil {
		me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, tag))
	}
}

func (me *FLV) onVideoPacket(pkt *av.Packet) {
	track := me.GetVideoTracks()[0]
	source := track.Source()

	switch pkt.Codec {
	case "AVC":
		if pkt.Get("Keyframe").(bool) && atomic.CompareAndSwapUint32(&me.readyState, format.RemuxWaiting, format.RemuxPumping) {
			me.Info.TimeBase = pkt.Timestamp
		}
		if source.GetInfoFrame() == nil || atomic.LoadUint32(&me.readyState) != format.RemuxPumping || (me.Mode&av.ModeKeyframe) == av.ModeKeyframe && !pkt.Get("Keyframe").(bool) {
			return
		}
	default:
		me.logger.Errorf("Unrecognized codec: %s", pkt.Codec)
		return
	}

	tag := me.format(pkt)
	if tag != nil {
		me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, tag))
	}
}

func (me *FLV) onError(e *ErrorEvent.ErrorEvent) {
	me.logger.Debugf(0, "%s: %s", e.Name, e.Message)
	me.Close()
}

func (me *FLV) onClose(e *Event.Event) {
	me.Close()
}

func (me *FLV) format(pkt *av.Packet) *av.Packet {
	backpointer := pkt.Length + 11
	tag := new(av.Packet).Init()
	tag.Kind = pkt.Kind
	tag.Codec = pkt.Codec
	tag.Length = pkt.Length
	tag.Timestamp = pkt.Timestamp - me.Info.TimeBase
	tag.StreamID = pkt.StreamID
	tag.Position = 0
	tag.Payload = make([]byte, backpointer+4)

	// header
	i := uint32(0)

	var kind byte
	switch pkt.Kind {
	case av.KindAudio:
		kind = KindAudio
	case av.KindVideo:
		kind = KindVideo
	case av.KindScript:
		kind = KindScript
	default:
		me.logger.Warnf("Unsupported packet kind %s by flv, ignored.")
		return nil
	}
	copy(tag.Payload[i:11], []byte{
		kind,
		byte(tag.Length >> 16), byte(tag.Length >> 8), byte(tag.Length),
		byte(tag.Timestamp >> 16), byte(tag.Timestamp >> 8), byte(tag.Timestamp), byte(tag.Timestamp >> 24),
		0x00, 0x00, 0x00, // Stream ID always 0.
	})
	i += 11

	// payload
	copy(tag.Payload[i:i+tag.Length], pkt.Payload)
	i += tag.Length

	// backpointer
	copy(tag.Payload[i:], []byte{
		byte(backpointer >> 24), byte(backpointer >> 16), byte(backpointer >> 8), byte(backpointer),
	})
	i += 4

	return tag
}

// SetDataFrame stores a data frame with the given key.
func (me *FLV) SetDataFrame(key string, pkt *av.Packet) {
	me.Info = *me.source.Information()
	me.MediaStream.SetDataFrame(key, pkt)
}

// Close detaches IRemuxer source, and closes IMediaStream.
func (me *FLV) Close() {
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
