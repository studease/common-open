package mux

import (
	"fmt"
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format/rtp"
	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/events"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
)

func init() {
	Register(TYPE_RTP, RTP{})
}

// RTP MUX
type RTP struct {
	events.EventDispatcher
	rtp.RTP

	logger     log.ILogger
	factory    log.ILoggerFactory
	stream     av.IReadableStream
	mode       uint32
	readyState uint32
	Duration   uint32
	Size       int64
	Frames     int64

	dataListener  *events.EventListener
	audioListener *events.EventListener
	videoListener *events.EventListener
	closeListener *events.EventListener

	// For DMX
	state  uint8
	packet rtp.Packet
}

// Init this class
func (me *RTP) Init(mode uint32, logger log.ILogger, factory log.ILoggerFactory) IMuxer {
	me.EventDispatcher.Init(logger)
	me.RTP.Init(logger, factory)
	me.logger = logger
	me.factory = factory
	me.mode = mode
	me.readyState = STATE_INITIALIZED
	me.dataListener = events.NewListener(me.onDataPacket, 0)
	me.audioListener = events.NewListener(me.onAudioPacket, 0)
	me.videoListener = events.NewListener(me.onVideoPacket, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// Attach the IReadableStream
func (me *RTP) Attach(stream av.IReadableStream) {
	if stream == nil {
		me.logger.Debugf(3, "Detaching stream")
		me.Close()
		return
	}

	if atomic.LoadUint32(&me.readyState) == STATE_INITIALIZED {
		panic("bad readyState")
	}

	me.stream = stream
	atomic.StoreUint32(&me.readyState, STATE_DETECTING)

	me.InfoFrame = stream.GetDataFrame("onMetaData")
	me.AudioInfoFrame = stream.AudioInfoFrame()
	me.VideoInfoFrame = stream.VideoInfoFrame()

	if me.InfoFrame != nil {
		me.onMetaData(me.InfoFrame.Value)
	}

	if me.AudioInfoFrame != nil && (me.mode&MODE_AUDIO) != 0 {
		me.AudioTrack().(*rtp.Track).Context.Parse(me.AudioInfoFrame)
		me.forward(me.AudioInfoFrame)
	}

	if me.VideoInfoFrame != nil && (me.mode&MODE_VIDEO) != 0 {
		me.VideoTrack().(*rtp.Track).Context.Parse(me.VideoInfoFrame)
	}

	me.stream.AddEventListener(MediaEvent.DATA, me.dataListener)
	me.stream.AddEventListener(MediaEvent.AUDIO, me.audioListener)
	me.stream.AddEventListener(MediaEvent.VIDEO, me.videoListener)
	me.stream.AddEventListener(Event.CLOSE, me.closeListener)
}

func (me *RTP) onDataPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.Handler == "@setDataFrame" {
		if pkt.Key == "onMetaData" {
			me.InfoFrame = pkt
			me.onMetaData(pkt.Value)
		}
	}
}

func (me *RTP) onAudioPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.DataType == aac.SPECIFIC_CONFIG {
		me.AudioInfoFrame = pkt
	}

	if (me.mode&MODE_AUDIO) == 0 || pkt.Length == 0 {
		me.Duration += pkt.Timestamp
		return
	}

	if pkt.DataType != aac.SPECIFIC_CONFIG {
		if me.AudioInfoFrame == nil || me.VideoInfoFrame != nil && atomic.LoadUint32(&me.readyState) <= STATE_DETECTING && (me.mode&MODE_ADVANCED) == 0 {
			return
		}
	}

	me.AudioTrack().(*rtp.Track).Context.Parse(pkt)
	me.forward(pkt)
}

func (me *RTP) onVideoPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.DataType == avc.SEQUENCE_HEADER {
		me.VideoInfoFrame = pkt
		return
	}

	if (me.mode&MODE_VIDEO) == 0 ||
		(me.mode&MODE_KEYFRAME) == MODE_KEYFRAME && pkt.FrameType != av.KEYFRAME && pkt.FrameType != av.GENERATED_KEYFRAME ||
		pkt.FrameType == av.INFO_OR_COMMAND_FRAME {
		me.Duration += pkt.Timestamp
		return
	}

	state := atomic.LoadUint32(&me.readyState)

	if state == STATE_DETECTING && (pkt.FrameType == av.KEYFRAME || pkt.FrameType == av.GENERATED_KEYFRAME) {
		atomic.StoreUint32(&me.readyState, STATE_ALIVE)
	}

	if me.VideoInfoFrame == nil {
		return
	}

	if state <= STATE_DETECTING {
		if (me.mode & MODE_ADVANCED) == 0 {
			return
		}

		// TODO: Generate keyframe
	}

	me.VideoTrack().(*rtp.Track).Context.Parse(pkt)
	me.forward(pkt)
}

// Append the data for demuxing
func (me *RTP) Append(data []byte) (int, error) {
	const (
		sw_v uint8 = iota
		sw_m
		sw_sn_0
		sw_sn_1
		sw_timestamp_0
		sw_timestamp_1
		sw_timestamp_2
		sw_timestamp_3
		sw_ssrc_0
		sw_ssrc_1
		sw_ssrc_2
		sw_ssrc_3
		sw_csrc_0
		sw_csrc_1
		sw_csrc_2
		sw_csrc_3
		sw_data
	)

	var (
		size = len(data)
	)

	for i := 0; i < size; i++ {
		ch := data[i]

		switch me.state {
		case sw_v:
			me.packet.V = ch >> 6
			me.packet.P = (ch >> 5) & 0x01
			me.packet.X = (ch >> 4) & 0x01
			me.packet.CC = ch & 0x0F
			me.state = sw_m

		case sw_m:
			me.packet.M = ch >> 7
			me.packet.PT = ch & 0x7F
			me.state = sw_sn_0

		case sw_sn_0:
			me.packet.SN = uint16(ch) << 8
			me.state = sw_sn_1

		case sw_sn_1:
			me.packet.SN |= uint16(ch)
			me.state = sw_timestamp_0

		case sw_timestamp_0:
			me.packet.Timestamp = uint32(ch) << 24
			me.state = sw_timestamp_1

		case sw_timestamp_1:
			me.packet.Timestamp |= uint32(ch) << 16
			me.state = sw_timestamp_1

		case sw_timestamp_2:
			me.packet.Timestamp |= uint32(ch) << 8
			me.state = sw_timestamp_1

		case sw_timestamp_3:
			me.packet.Timestamp |= uint32(ch)
			me.state = sw_data

		case sw_ssrc_0:
			me.packet.SSRC = uint32(ch) << 24
			me.state = sw_ssrc_1

		case sw_ssrc_1:
			me.packet.SSRC |= uint32(ch) << 16
			me.state = sw_ssrc_2

		case sw_ssrc_2:
			me.packet.SSRC |= uint32(ch) << 8
			me.state = sw_ssrc_3

		case sw_ssrc_3:
			me.packet.SSRC |= uint32(ch)
			me.state = sw_csrc_0

		case sw_csrc_0:
			n := uint32(ch) << 24
			me.packet.CSRC = append(me.packet.CSRC, n)
			me.state = sw_csrc_1

		case sw_csrc_1:
			n := len(me.packet.CSRC) - 1
			me.packet.CSRC[n] |= uint32(ch) << 16
			me.state = sw_csrc_2

		case sw_csrc_2:
			n := len(me.packet.CSRC) - 1
			me.packet.CSRC[n] |= uint32(ch) << 8
			me.state = sw_csrc_3

		case sw_csrc_3:
			n := len(me.packet.CSRC) - 1
			me.packet.CSRC[n] |= uint32(ch)
			me.state = sw_data

		case sw_data:
			// TODO: Parse payload

		default:
			return i, fmt.Errorf("unrecognized state")
		}
	}

	return size, nil
}

func (me *RTP) forward(raw *av.Packet) {
	var (
		event string
	)

	switch raw.Type {
	case av.TYPE_AUDIO:
		event = MediaEvent.AUDIO
	case av.TYPE_VIDEO:
		event = MediaEvent.VIDEO
	default:
		event = MediaEvent.DATA
	}

	arr := me.Format(raw)

	for _, old := range arr {
		pkt := raw.Clone(old.Format())
		me.DispatchEvent(MediaEvent.New(event, me, pkt))
		me.Size += int64(pkt.Length)
	}

	me.Duration += raw.Timestamp
	me.Frames++
}

func (me *RTP) onMetaData(o *amf.Value) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(3, "Failed to handle onMetaData: %v", err)
		}
	}()

	onMetaData(me.Information(), o)
}

func (me *RTP) onClose(e *Event.Event) {
	me.Close()
}

// Stream returns the attached IReadableStream
func (me *RTP) Stream() av.IReadableStream {
	return me.stream
}

// Mode returns the mode of this muxer
func (me *RTP) Mode() uint32 {
	return me.mode
}

// ReadyState returns the readyState of this muxer
func (me *RTP) ReadyState() uint32 {
	return atomic.LoadUint32(&me.readyState)
}

// Close this muxer
func (me *RTP) Close() {
	switch atomic.LoadUint32(&me.readyState) {
	case STATE_DETECTING:
		fallthrough
	case STATE_ALIVE:
		atomic.StoreUint32(&me.readyState, STATE_CLOSING)

		me.stream.RemoveEventListener(MediaEvent.DATA, me.dataListener)
		me.stream.RemoveEventListener(MediaEvent.AUDIO, me.audioListener)
		me.stream.RemoveEventListener(MediaEvent.VIDEO, me.videoListener)
		me.stream.RemoveEventListener(Event.CLOSE, me.closeListener)
		me.DispatchEvent(Event.New(Event.CLOSE, me))

		me.RTP.Close()
		me.stream = nil
		atomic.StoreUint32(&me.readyState, STATE_CLOSED)
		me.Duration = 0
		me.Size = 0
		me.Frames = 0
	}
}
