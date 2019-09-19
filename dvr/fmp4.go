package dvr

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/events"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
	"github.com/studease/common/mux"
	basecfg "github.com/studease/common/utils/config"
)

func init() {
	Register(TYPE_FMP4, FMP4{})
}

// FMP4 DVR
type FMP4 struct {
	cfg         *basecfg.DVR
	logger      log.ILogger
	factory     log.ILoggerFactory
	mtx         sync.RWMutex
	mux         mux.FMP4
	segmer      av.ISegmableStream
	audioBuffer av.ISourceBuffer
	videoBuffer av.ISourceBuffer
	parameters  string
	directory   string
	closeNotify chan bool

	audioListener *events.EventListener
	videoListener *events.EventListener
	closeListener *events.EventListener
}

// Init this class
func (me *FMP4) Init(cfg *basecfg.DVR, logger log.ILogger, factory log.ILoggerFactory) IDVR {
	me.mux.Init(mux.Mode(cfg.Mode, ","), logger, factory)
	me.segmer = new(SegmableStream).Init()
	me.cfg = cfg
	me.logger = logger
	me.factory = factory
	me.closeNotify = make(chan bool, 1)
	me.audioListener = events.NewListener(me.onAudioPacket, 0)
	me.videoListener = events.NewListener(me.onVideoPacket, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// Attach the IReadableStream
func (me *FMP4) Attach(stream av.IReadableStream) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}
	}()

	if stream == nil {
		me.logger.Debugf(3, "DVR detaching stream")
		me.Close()
		return
	}

	now := time.Now()

	me.parameters = stream.Parameters()
	me.directory = now.Format(me.cfg.Directory + "/" + me.cfg.FileName)
	me.directory = strings.Replace(me.directory, "${APPLICATION}", stream.AppName(), -1)
	me.directory = strings.Replace(me.directory, "${INSTANCE}", stream.InstName(), -1)
	me.directory = strings.Replace(me.directory, "${STREAM}", stream.Name(), -1)
	if me.cfg.Unique {
		me.directory += fmt.Sprintf("-%d", now.Unix())
	}

	os.RemoveAll(me.directory)

	err := os.MkdirAll(me.directory, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	me.mux.AddEventListener(MediaEvent.AUDIO, me.audioListener)
	me.mux.AddEventListener(MediaEvent.VIDEO, me.videoListener)
	me.mux.AddEventListener(Event.CLOSE, me.closeListener)
	me.mux.Attach(stream)
}

func (me *FMP4) onAudioPacket(e *MediaEvent.MediaEvent) {
	if e.Packet.DataType == aac.SPECIFIC_CONFIG {
		data := e.Packet.Payload

		me.audioBuffer = me.segmer.AddSourceBuffer(av.KIND_AUDIO, e.Packet.Codec)
		me.audioBuffer.Write(me.directory+"/audio_init.m4s", data)

		if (me.mux.Mode()&mux.MODE_VIDEO) != 0 && me.mux.VideoInfoFrame != nil {
			data = me.mux.GetInitSegment()
		}

		me.audioBuffer.Write(me.directory+"/init.m4s", data)
		return
	}

	b := me.audioBuffer.(*SourceBuffer)
	if b == nil {
		return
	}

	b.Append(e.Packet)
}

func (me *FMP4) onVideoPacket(e *MediaEvent.MediaEvent) {
	var (
		a, v interface{}
	)

	if e.Packet.DataType == avc.SEQUENCE_HEADER {
		data := e.Packet.Payload

		me.videoBuffer = me.segmer.AddSourceBuffer(av.KIND_VIDEO, e.Packet.Codec)
		me.videoBuffer.Write(me.directory+"/video_init.m4s", data)

		if (me.mux.Mode()&mux.MODE_AUDIO) != 0 && me.mux.AudioInfoFrame != nil {
			data = me.mux.GetInitSegment()
		}

		me.videoBuffer.Write(me.directory+"/init.m4s", data)
		return
	}

	b := me.videoBuffer.(*SourceBuffer)
	if b == nil {
		return
	}

	if (e.Packet.FrameType == av.KEYFRAME || e.Packet.FrameType == av.GENERATED_KEYFRAME) && e.Packet.DataType == avc.NALU &&
		(me.cfg.MaxDuration > 0 && b.Duration >= me.cfg.MaxDuration ||
			me.cfg.MaxSize > 0 && b.Size >= me.cfg.MaxSize ||
			me.cfg.MaxFrames > 0 && b.Frames >= me.cfg.MaxFrames) {
		name := fmt.Sprintf("/video_%d.m4s", me.videoBuffer.Timestamp())
		v, _ = me.videoBuffer.Write(me.directory+name, me.videoBuffer.Bytes())

		if me.audioBuffer != nil {
			name = fmt.Sprintf("/audio_%d.m4s", me.audioBuffer.Timestamp())
			a, _ = me.audioBuffer.Write(me.directory+name, me.videoBuffer.Bytes())
		}

		me.segmer.AddPoint(v.(*Segment).Timestamp, new(segEntry).Init(a.(*Segment), v.(*Segment)))
	}

	b.Append(e.Packet)
}

func (me *FMP4) onClose(e *Event.Event) {
	if me.mux.ReadyState() <= mux.STATE_CLOSING {
		me.closeNotify <- true
	}
}

// CloseNotify returns a channel that receives at most a single value (true) when the client connection has gone away
func (me *FMP4) CloseNotify() <-chan bool {
	return me.closeNotify
}

// Close this DVR
func (me *FMP4) Close() {
	switch me.mux.ReadyState() {
	case mux.STATE_DETECTING:
		fallthrough
	case mux.STATE_ALIVE:
		me.mux.Close()

	case mux.STATE_CLOSING:
		me.mux.RemoveEventListener(MediaEvent.AUDIO, me.audioListener)
		me.mux.RemoveEventListener(MediaEvent.VIDEO, me.videoListener)
		me.mux.RemoveEventListener(Event.CLOSE, me.closeListener)

		if b := me.videoBuffer; b != nil && b.Len() > 0 {
			name := fmt.Sprintf("/video_%d.m4s", b.Timestamp())
			b.Write(me.directory+name, b.Bytes())
			b.Clear()
		}
		if b := me.audioBuffer; b != nil && b.Len() > 0 {
			name := fmt.Sprintf("/audio_%d.m4s", b.Timestamp())
			b.Write(me.directory+name, b.Bytes())
			b.Clear()
		}

		me.segmer.Clear()
		close(me.closeNotify)
	}
}
