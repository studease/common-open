package dvr

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/events"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
	"github.com/studease/common/mux"
	"github.com/studease/common/target"
	basecfg "github.com/studease/common/utils/config"
)

func init() {
	Register(TYPE_FLV, FLV{})
}

// FLV DVR
type FLV struct {
	cfg         *basecfg.DVR
	logger      log.ILogger
	factory     log.ILoggerFactory
	mtx         sync.RWMutex
	mux         mux.FLV
	seeker      av.ISeekableStream
	parameters  string
	directory   string
	filename    string
	file        *os.File
	closeNotify chan bool

	dataListener  *events.EventListener
	audioListener *events.EventListener
	videoListener *events.EventListener
	closeListener *events.EventListener
}

// Init this class
func (me *FLV) Init(cfg *basecfg.DVR, logger log.ILogger, factory log.ILoggerFactory) IDVR {
	me.mux.Init(mux.Mode(cfg.Mode, ","), logger, factory)
	me.cfg = cfg
	me.logger = logger
	me.factory = factory
	me.seeker = new(SeekableStream).Init()
	me.closeNotify = make(chan bool, 1)
	me.dataListener = events.NewListener(me.onDataPacket, 0)
	me.audioListener = events.NewListener(me.onAudioPacket, 0)
	me.videoListener = events.NewListener(me.onVideoPacket, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// Attach the IReadableStream
func (me *FLV) Attach(stream av.IReadableStream) {
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
	me.directory = now.Format(me.cfg.Directory)
	me.directory = strings.Replace(me.directory, "${APPLICATION}", stream.AppName(), -1)
	me.directory = strings.Replace(me.directory, "${INSTANCE}", stream.InstName(), -1)

	err := os.MkdirAll(me.directory, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	err = me.open(now, stream.Name())
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	me.mux.AddEventListener(MediaEvent.DATA, me.dataListener)
	me.mux.AddEventListener(MediaEvent.AUDIO, me.audioListener)
	me.mux.AddEventListener(MediaEvent.VIDEO, me.videoListener)
	me.mux.AddEventListener(Event.CLOSE, me.closeListener)
	me.mux.Attach(stream)
}

func (me *FLV) open(now time.Time, stream string) error {
	extension := ".flv"
	if me.cfg.Unique {
		extension = fmt.Sprintf("-%d.flv", now.Unix())
	}

	me.filename = now.Format(me.cfg.FileName)
	me.filename = strings.Replace(me.filename, "${STREAM}", stream, -1) + extension

	name := me.directory + "/" + me.filename

	perm := os.O_RDWR | os.O_CREATE
	if me.cfg.Append {
		perm |= os.O_APPEND
	} else {
		perm |= os.O_TRUNC
	}

	f, err := os.OpenFile(name, perm, 0666)
	if err != nil {
		return err
	}

	old := me.file
	ptr := unsafe.Pointer(me.file)
	atomic.StorePointer(&ptr, unsafe.Pointer(f))
	old.Close()

	if url := &me.cfg.OnRecord; url.Enable {
		go me.sendNotification(url, "onRecord", name)
	}

	return nil
}

func (me *FLV) onDataPacket(e *MediaEvent.MediaEvent) {
	_, err := me.file.Write(e.Packet.Payload)
	if err != nil {
		me.logger.Debugf(3, "DVR failed to write: %v", err)
		me.Close()
	}
}

func (me *FLV) onAudioPacket(e *MediaEvent.MediaEvent) {
	_, err := me.file.Write(e.Packet.Payload)
	if err != nil {
		me.logger.Debugf(3, "DVR failed to write: %v", err)
		me.Close()
	}
}

func (me *FLV) onVideoPacket(e *MediaEvent.MediaEvent) {
	if (e.Packet.FrameType == av.KEYFRAME || e.Packet.FrameType == av.GENERATED_KEYFRAME) && e.Packet.DataType == avc.NALU {
		if me.cfg.MaxDuration > 0 && me.mux.Duration >= me.cfg.MaxDuration ||
			me.cfg.MaxSize > 0 && me.mux.Size >= me.cfg.MaxSize ||
			me.cfg.MaxFrames > 0 && me.mux.Frames >= me.cfg.MaxFrames {

			if !me.cfg.Unique {
				me.Close()
				return
			}

			err := me.open(time.Now(), me.mux.Stream().Name())
			if err != nil {
				me.logger.Errorf("Failed to rotate file: %v", err)
				me.Close()
				return
			}

			return
		}

		me.seeker.AddPoint(me.mux.Duration, me.mux.Size)
	}

	_, err := me.file.Write(e.Packet.Payload)
	if err != nil {
		me.logger.Debugf(3, "DVR failed to write: %v", err)
		me.Close()
	}
}

func (me *FLV) onClose(e *Event.Event) {
	if me.mux.ReadyState() <= mux.STATE_CLOSING {
		me.closeNotify <- true

		if url := &me.cfg.OnRecordDone; url.Enable {
			name := me.directory + "/" + me.filename
			go me.sendNotification(url, "onRecordDone", name)
		}
	}
}

func (me *FLV) sendNotification(url *basecfg.URL, event string, name string) {
	i := strings.Index(name, "/")
	if i == -1 {
		i = 0
	}

	path := []byte(name)[i:]

	rawquery := "call=" + event
	rawquery += "&dvr=" + me.cfg.ID
	rawquery += "&path=" + string(path)

	if me.parameters != "" {
		rawquery += "&" + me.parameters
	}

	res, err := target.Request(url, rawquery)
	if err != nil {
		me.logger.Warnf("Failed to send %s notification: %v", event, err)
		return
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		me.logger.Debugf(3, "Sent event: type=%s, url=%s?%s", event, url.Path, rawquery)
	default:
		me.logger.Warnf("Failed to sent event: type=%s, url=%s?%s, %v", event, url.Path, rawquery, err)
	}
}

// CloseNotify returns a channel that receives at most a single value (true) when the client connection has gone away
func (me *FLV) CloseNotify() <-chan bool {
	return me.closeNotify
}

// Close this DVR
func (me *FLV) Close() {
	switch me.mux.ReadyState() {
	case mux.STATE_DETECTING:
		fallthrough
	case mux.STATE_ALIVE:
		me.mux.Close()

	case mux.STATE_CLOSING:
		me.mux.RemoveEventListener(MediaEvent.DATA, me.dataListener)
		me.mux.RemoveEventListener(MediaEvent.AUDIO, me.audioListener)
		me.mux.RemoveEventListener(MediaEvent.VIDEO, me.videoListener)
		me.mux.RemoveEventListener(Event.CLOSE, me.closeListener)

		me.seeker.Clear()
		close(me.closeNotify)
	}
}
