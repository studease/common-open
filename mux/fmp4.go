package mux

import (
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format/mp4"
	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/events"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
)

func init() {
	Register(TYPE_FMP4, FMP4{})
}

// FMP4 MUX
type FMP4 struct {
	events.EventDispatcher
	mp4.MP4

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
}

// Init this class
func (me *FMP4) Init(mode uint32, logger log.ILogger, factory log.ILoggerFactory) IMuxer {
	me.EventDispatcher.Init(logger)
	me.MP4.Init(logger, factory)
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
func (me *FMP4) Attach(stream av.IReadableStream) {
	if stream == nil {
		me.logger.Debugf(3, "MUX detaching stream")
		me.Close()
		return
	}

	if atomic.LoadUint32(&me.readyState) != STATE_INITIALIZED {
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
		track := me.NewTrack(me.AudioInfoFrame.Codec)
		track.Context.Parse(me.AudioInfoFrame)
		me.forward(me.AudioInfoFrame)
	}

	if me.VideoInfoFrame != nil && (me.mode&MODE_VIDEO) != 0 {
		track := me.NewTrack(me.VideoInfoFrame.Codec)
		track.Context.Parse(me.VideoInfoFrame)
		me.forward(me.VideoInfoFrame)
	}

	me.stream.AddEventListener(MediaEvent.DATA, me.dataListener)
	me.stream.AddEventListener(MediaEvent.AUDIO, me.audioListener)
	me.stream.AddEventListener(MediaEvent.VIDEO, me.videoListener)
	me.stream.AddEventListener(Event.CLOSE, me.closeListener)
}

func (me *FMP4) onDataPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.Handler == "@setDataFrame" {
		if pkt.Key == "onMetaData" {
			me.InfoFrame = pkt
			me.onMetaData(pkt.Value)
		}
	}
}

func (me *FMP4) onAudioPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.DataType == aac.SPECIFIC_CONFIG {
		me.AudioInfoFrame = pkt

		track := me.NewTrack(me.AudioInfoFrame.Codec)
		track.Context.Parse(me.AudioInfoFrame)
	}

	if (me.mode&MODE_AUDIO) == 0 || pkt.Length == 0 {
		me.Duration += pkt.Timestamp
		return
	}

	if pkt.DataType != aac.SPECIFIC_CONFIG {
		if me.AudioInfoFrame == nil || me.VideoInfoFrame != nil && atomic.LoadUint32(&me.readyState) <= STATE_DETECTING && (me.mode&MODE_ADVANCED) == 0 {
			return
		}

		me.AudioTrack().(*mp4.Track).Context.Parse(pkt)
	}

	me.forward(pkt)
}

func (me *FMP4) onVideoPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.DataType == avc.SEQUENCE_HEADER {
		me.VideoInfoFrame = pkt

		track := me.NewTrack(me.VideoInfoFrame.Codec)
		track.Context.Parse(me.VideoInfoFrame)
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

	if pkt.DataType != avc.SEQUENCE_HEADER {
		if me.VideoInfoFrame == nil {
			return
		}

		if state <= STATE_DETECTING {
			if (me.mode & MODE_ADVANCED) == 0 {
				return
			}

			// TODO: Generate keyframe
		}

		me.VideoTrack().(*mp4.Track).Context.Parse(pkt)
	}

	me.forward(pkt)
}

// Append the data for demuxing
func (me *FMP4) Append(data []byte) (int, error) {
	// TODO: Parse fmp4 stream
	return 0, nil
}

func (me *FMP4) forward(raw *av.Packet) {
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

	seg := me.Format(raw)
	pkt := raw.Clone(seg)

	me.DispatchEvent(MediaEvent.New(event, me, pkt))
	me.Duration += raw.Timestamp
	me.Size += int64(pkt.Length)
	me.Frames++
}

func (me *FMP4) onMetaData(o *amf.Value) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(3, "Failed to handle onMetaData: %v", err)
		}
	}()

	onMetaData(me.Information(), o)
}

func (me *FMP4) onClose(e *Event.Event) {
	me.Close()
}

// Stream returns the attached IReadableStream
func (me *FMP4) Stream() av.IReadableStream {
	return me.stream
}

// Mode returns the mode of this muxer
func (me *FMP4) Mode() uint32 {
	return me.mode
}

// ReadyState returns the readyState of this muxer
func (me *FMP4) ReadyState() uint32 {
	return atomic.LoadUint32(&me.readyState)
}

// Close this muxer
func (me *FMP4) Close() {
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

		me.MP4.Close()
		me.stream = nil
		atomic.StoreUint32(&me.readyState, STATE_CLOSED)
		me.Duration = 0
		me.Size = 0
		me.Frames = 0
	}
}
