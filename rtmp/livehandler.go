package rtmp

import (
	"bytes"
	"strings"
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/events"
	CommandEvent "github.com/studease/common/events/commandevent"
	Event "github.com/studease/common/events/event"
	MediaRecorderEvent "github.com/studease/common/events/mediarecorderevent"
	Code "github.com/studease/common/events/netstatusevent/code"
	Level "github.com/studease/common/events/netstatusevent/level"
	"github.com/studease/common/log"
	rtmpcfg "github.com/studease/common/rtmp/config"
	"github.com/studease/common/rtmp/message"
	"github.com/studease/common/rtmp/message/command"
	CSID "github.com/studease/common/rtmp/message/csid"
	EventType "github.com/studease/common/rtmp/message/eventtype"
	LimitType "github.com/studease/common/rtmp/message/limittype"
	"github.com/studease/common/target"
	basecfg "github.com/studease/common/utils/config"
)

// Object encodings.
const (
	AMF0 byte = 0
	AMF3 byte = 3
)

var (
	fmsProperties = amf.NewValue(amf.OBJECT)
	fmsEncoding   = amf.NewValue(amf.DOUBLE).Set("objectEncoding", float64(AMF0))
	fmsVersion    = amf.NewValue(amf.ECMA_ARRAY)
)

func init() {
	Register("rtmp-live", LiveHandler{})

	fmsProperties.Add(amf.NewValue(amf.STRING).Set("fmsVer", "FMS/5,0,3,3029"))
	fmsProperties.Add(amf.NewValue(amf.DOUBLE).Set("capabilities", float64(255)))
	fmsProperties.Add(amf.NewValue(amf.DOUBLE).Set("mode", float64(1)))

	fmsVersion.Key = "data"
	fmsVersion.Add(amf.NewValue(amf.STRING).Set("version", "5,0,3,3029"))
}

// LiveHandler provides live broadcast service.
type LiveHandler struct {
	srv     *Server
	cfg     *rtmpcfg.Location
	logger  log.ILogger
	factory log.ILoggerFactory

	connectListener      *events.EventListener
	createStreamListener *events.EventListener
	publishListener      *events.EventListener
	recorderListener     *events.EventListener
	playListener         *events.EventListener
	seekListener         *events.EventListener
	pauseListener        *events.EventListener
	closeStreamListener  *events.EventListener
	closeListener        *events.EventListener
}

// Init this class.
func (me *LiveHandler) Init(srv *Server, cfg *rtmpcfg.Location, logger log.ILogger, factory log.ILoggerFactory) IHandler {
	me.srv = srv
	me.cfg = cfg
	me.logger = logger
	me.factory = factory
	me.connectListener = events.NewListener(me.onConnect, 0)
	me.createStreamListener = events.NewListener(me.onCreateStream, 0)
	me.publishListener = events.NewListener(me.onPublish, 0)
	me.recorderListener = events.NewListener(me.onRecorderEvent, 0)
	me.playListener = events.NewListener(me.onPlay, 0)
	me.seekListener = events.NewListener(me.onSeek, 0)
	me.pauseListener = events.NewListener(me.onPause, 0)
	me.closeStreamListener = events.NewListener(me.onCloseStream, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// ServeRTMP handles the NetConnection.
func (me *LiveHandler) ServeRTMP(nc *NetConnection) error {
	nc.AddEventListener(CommandEvent.CONNECT, me.connectListener)
	nc.AddEventListener(CommandEvent.CREATE_STREAM, me.createStreamListener)
	nc.AddEventListener(Event.CLOSE, me.closeListener)
	return nil
}

func (me *LiveHandler) onConnect(e *CommandEvent.CommandEvent) {
	nc := e.Target.(*NetConnection)
	m := e.Message

	if url := &me.cfg.OnOpen; url.Enable {
		err := nc.sendNotification(url, "connect")
		if err != nil {
			me.logger.Errorf("Failed to send \"connect\" notification: %v", err)
			me.srv.Reject(nc, err.Error())
			return
		}
	}
	me.srv.Accept(nc)

	nc.SetAckWindowSize(DEFAULT_ACK_WINDOW_SIZE)
	nc.SetPeerBandwidth(DEFAULT_PEER_BANDWIDTH, LimitType.DYNAMIC)
	nc.SendUserControl(EventType.STREAM_BEGIN, 0, 0, 0)
	nc.SetChunkSize(DEFAULT_CHUNK_SIZE)

	info := NewInfoObject(Level.STATUS, Code.NETCONNECTION_CONNECT_SUCCESS, "connect success")
	info.Add(fmsEncoding)
	info.Add(fmsVersion)

	var b bytes.Buffer
	amf.EncodeString(&b, command.RESULT)
	amf.EncodeDouble(&b, 1)
	amf.Encode(&b, fmsProperties)
	amf.Encode(&b, info)

	_, err := nc.sendBytes(CSID.COMMAND, message.COMMAND, 0, 0, b.Bytes())
	if err != nil {
		me.logger.Errorf("Failed to reply command \"%s\" with \"%s\"", m.CommandName, command.RESULT)
		nc.Close()
		return
	}
}

func (me *LiveHandler) onCreateStream(e *CommandEvent.CommandEvent) {
	nc := e.Target.(*NetConnection)
	m := e.Message

	ns := new(NetStream).Init(nc, me.logger, me.factory)
	ns.AddEventListener(CommandEvent.PUBLISH, me.publishListener)
	ns.AddEventListener(CommandEvent.PLAY, me.playListener)
	ns.AddEventListener(CommandEvent.CLOSE_STREAM, me.closeStreamListener)

	var b bytes.Buffer
	amf.EncodeString(&b, command.RESULT)
	amf.EncodeDouble(&b, float64(m.TransactionID))
	amf.EncodeNull(&b)
	amf.EncodeDouble(&b, float64(ns.id))

	_, err := nc.sendBytes(CSID.COMMAND, message.COMMAND, 0, 0, b.Bytes())
	if err != nil {
		me.logger.Errorf("Failed to reply command \"%s\": %v", m.CommandName, err)
		nc.Close()
	}
}

func (me *LiveHandler) onPublish(e *CommandEvent.CommandEvent) {
	ns := e.Target.(*NetStream)
	nc := ns.nc
	m := e.Message

	stream := me.srv.GetStream(nc.AppName, nc.InstName, m.PublishingName)
	if stream == nil {
		me.logger.Errorf("Failed to get stream")
		ns.SendStatus(Level.ERROR, Code.NETSTREAM_FAILED, "internal error")
		nc.Close()
		return
	}

	if (atomic.LoadUint32(&stream.readyState) & STREAM_PUBLISHING) != 0 {
		ns.SendStatus(Level.ERROR, Code.NETSTREAM_PUBLISH_BADNAME, "publish bad name")
		nc.Close()
		return
	}

	if url := &me.cfg.OnPublish; url.Enable {
		err := ns.sendNotification(url, "publish")
		if err != nil {
			me.logger.Errorf("Failed to send \"publish\" notification: %v", err)
			ns.SendStatus(Level.ERROR, Code.NETSTREAM_PLAY_FAILED, err.Error())
			ns.Close()
			return
		}
	}

	err := ns.SendStatus(Level.STATUS, Code.NETSTREAM_PUBLISH_START, "publish start")
	if err != nil {
		me.logger.Errorf("Failed to send status: %s", Code.NETSTREAM_PUBLISH_START)
		nc.Close()
		return
	}

	atomic.StoreUint32(&ns.readyState, STREAM_PUBLISHING)
	ns.Sink(stream)

	// Start IMediaRecorder
	for _, cfg := range me.cfg.DVRs {
		constraints := new(av.MediaRecorderConstraints)
		constraints.Mode = av.Mode(cfg.Mode, ",")
		constraints.Directory = stream.Info.StartTime.Format(cfg.Directory)
		constraints.Directory = strings.Replace(constraints.Directory, "${APPLICATION}", nc.AppName, -1)
		constraints.Directory = strings.Replace(constraints.Directory, "${INSTANCE}", nc.InstName, -1)
		constraints.FileName = strings.Replace(cfg.FileName, "${STREAM}", stream.Name(), -1)
		constraints.Unique = cfg.Unique
		constraints.Append = cfg.Append
		constraints.Chunks = cfg.Chunks
		constraints.Segments = cfg.Segments
		constraints.MaxDuration = cfg.MaxDuration
		constraints.MaxSize = cfg.MaxSize
		constraints.MaxFrames = cfg.MaxFrames

		recorder := stream.NewRecorder(cfg.Name, constraints, me.factory)
		recorder.AddEventListener(MediaRecorderEvent.START, me.recorderListener)
		recorder.AddEventListener(MediaRecorderEvent.PAUSE, me.recorderListener)
		recorder.AddEventListener(MediaRecorderEvent.RESUME, me.recorderListener)
		recorder.AddEventListener(MediaRecorderEvent.STOP, me.recorderListener)
		recorder.Source(stream)
		if (constraints.Mode & av.ModeOff) == 0 {
			recorder.Start()
		}
	}

	// Publish to proxy
	if url := &me.cfg.Proxy; url.Enable && !m.Flag {
		u, err := target.Parse(url.Path)
		if err != nil {
			me.logger.Warnf("Failed to parse url: %v", err)
			return
		}

		u = strings.Replace(u, "${APPLICATION}", nc.AppName, -1)
		u = strings.Replace(u, "${INSTANCE}", nc.InstName, -1)
		u = strings.Replace(u, "${STREAM}", stream.Name(), -1)

		ps := new(Proxy).Init(u, me.srv, me.factory.NewLogger("PROXY"), me.factory)
		ps.AddEventListener(CommandEvent.CREATE_STREAM, me.createStreamListener)

		err = ps.Publish(stream)
		if err != nil {
			me.logger.Warnf("Failed to publish to proxy: %v", err)
			ps.Close()
			return
		}
	}
}

func (me *LiveHandler) onRecorderEvent(e *MediaRecorderEvent.MediaRecorderEvent) {
	recorder := e.Target.(av.IMediaRecorder)
	me.logger.Debugf(4, "MediaRecorder.on%s", e.Type)

	// TODO(spencerlau): Send notification.
	switch e.Type {
	case MediaRecorderEvent.STOP:
		recorder.RemoveEventListener(MediaRecorderEvent.START, me.recorderListener)
		recorder.RemoveEventListener(MediaRecorderEvent.PAUSE, me.recorderListener)
		recorder.RemoveEventListener(MediaRecorderEvent.RESUME, me.recorderListener)
		recorder.RemoveEventListener(MediaRecorderEvent.STOP, me.recorderListener)
	}
}

func (me *LiveHandler) onPlay(e *CommandEvent.CommandEvent) {
	ns := e.Target.(*NetStream)
	nc := ns.nc
	m := e.Message

	if url := &me.cfg.OnPlay; url.Enable {
		err := ns.sendNotification(url, "play")
		if err != nil {
			me.logger.Errorf("Failed to send \"play\" notification: %v", err)
			ns.SendStatus(Level.ERROR, Code.NETSTREAM_PLAY_FAILED, err.Error())
			ns.Close()
			return
		}
	}

	stream := me.srv.GetStream(nc.AppName, nc.InstName, m.StreamName)
	if stream == nil {
		me.logger.Errorf("Failed to get stream")
		ns.SendStatus(Level.ERROR, Code.NETSTREAM_FAILED, "internal error")
		nc.Close()
		return
	}

	err := nc.SendUserControl(EventType.STREAM_BEGIN, ns.id, 0, 0)
	if err != nil {
		me.logger.Errorf("Failed to send user control: event=0x%02X, stream=%d", EventType.STREAM_BEGIN, ns.id)
		nc.Close()
		return
	}

	if m.Reset {
		err = ns.SendStatus(Level.STATUS, Code.NETSTREAM_PLAY_RESET, "play reset")
		if err != nil {
			me.logger.Errorf("Failed to send status: %s", Code.NETSTREAM_PLAY_RESET)
			nc.Close()
			return
		}
	}

	err = ns.SendStatus(Level.STATUS, Code.NETSTREAM_PLAY_START, "play start")
	if err != nil {
		me.logger.Errorf("Failed to send status: %s", Code.NETSTREAM_PLAY_START)
		nc.Close()
		return
	}

	ns.AddEventListener(CommandEvent.SEEK, me.seekListener)
	ns.AddEventListener(CommandEvent.PAUSE, me.pauseListener)
	ns.Source(stream)

	// Play from proxy
	if url := &me.cfg.Proxy; (atomic.LoadUint32(&stream.readyState)&STREAM_PUBLISHING) == 0 && url.Enable {
		u, err := target.Parse(url.Path)
		if err != nil {
			me.logger.Warnf("Failed to parse url: %v", err)
			return
		}

		u = strings.Replace(u, "${APPLICATION}", nc.AppName, -1)
		u = strings.Replace(u, "${INSTANCE}", nc.InstName, -1)
		u = strings.Replace(u, "${STREAM}", stream.Name(), -1)

		ps := new(Proxy).Init(u, me.srv, me.factory.NewLogger("PROXY"), me.factory)
		ps.AddEventListener(CommandEvent.CREATE_STREAM, me.createStreamListener)

		err = ps.Play(m.StreamName)
		if err != nil {
			me.logger.Warnf("Failed to play from proxy: %v", err)
			ns.SendStatus(Level.ERROR, Code.NETSTREAM_PLAY_FAILED, "bad proxy")
			ps.Close()
			return
		}
	} else {
		// ns.SendStatus(Level.ERROR, Code.NETSTREAM_PLAY_STREAMNOTFOUND, "stream not found")
		// nc.Close()
	}
}

func (me *LiveHandler) onSeek(e *CommandEvent.CommandEvent) {
	ns := e.Target.(*NetStream)
	nc := ns.nc

	info := NewInfoObject(Level.ERROR, Code.NETSTREAM_SEEK_FAILED, "not allowed")

	var b bytes.Buffer
	amf.EncodeString(&b, command.ERROR)
	amf.EncodeDouble(&b, 0)
	amf.EncodeNull(&b)
	amf.EncodeObject(&b, info)

	_, err := nc.sendBytes(CSID.COMMAND_2, message.COMMAND, 0, ns.id, b.Bytes())
	if err != nil {
		me.logger.Errorf("Failed to send status: %s", Code.NETSTREAM_SEEK_FAILED)
		nc.Close()
		return
	}
}

func (me *LiveHandler) onPause(e *CommandEvent.CommandEvent) {
	ns := e.Target.(*NetStream)
	nc := ns.nc
	m := e.Message

	ns.pause = m.Flag
	ns.time = m.MilliSeconds

	info := NewInfoObject(Level.ERROR, Code.NETSTREAM_FAILED, "not allowed")

	var b bytes.Buffer
	amf.EncodeString(&b, command.ERROR)
	amf.EncodeDouble(&b, 0)
	amf.EncodeNull(&b)
	amf.EncodeObject(&b, info)

	_, err := nc.sendBytes(CSID.COMMAND_2, message.COMMAND, 0, ns.id, b.Bytes())
	if err != nil {
		me.logger.Errorf("Failed to send status: %s", Code.NETSTREAM_FAILED)
		nc.Close()
		return
	}
}

func (me *LiveHandler) onCloseStream(e *CommandEvent.CommandEvent) {
	var (
		url   *basecfg.URL
		event string
	)

	ns := e.Target.(*NetStream)

	switch atomic.LoadUint32(&ns.readyState) {
	case STREAM_UNPUBLISHING:
		url = &me.cfg.OnPublishDone
		event = "unpublish"
	case STREAM_UNPLAYING:
		url = &me.cfg.OnPlayDone
		event = "unplay"
	default:
		me.logger.Warnf("Bad state!")
		return
	}

	if url.Enable {
		err := ns.sendNotification(url, event)
		if err != nil {
			me.logger.Errorf("Failed to send \"%s\" notification: %v", event, err)
			return
		}
	}
}

func (me *LiveHandler) onClose(e *Event.Event) {
	nc := e.Target.(*NetConnection)

	if url := &me.cfg.OnClose; url.Enable {
		err := nc.sendNotification(url, "close")
		if err != nil {
			me.logger.Errorf("Failed to send \"close\" notification: %v", err)
			return
		}
	}
}

// NewInfoObject creates a rtmp info object.
func NewInfoObject(level string, code string, description string) *amf.Value {
	info := amf.NewValue(amf.OBJECT)
	info.Add(amf.NewValue(amf.STRING).Set("level", level))
	info.Add(amf.NewValue(amf.STRING).Set("code", code))
	info.Add(amf.NewValue(amf.STRING).Set("description", description))
	return info
}
