package rtmp

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/dvr"
	"github.com/studease/common/events"
	CommandEvent "github.com/studease/common/events/commandevent"
	Code "github.com/studease/common/events/netstatusevent/code"
	Level "github.com/studease/common/events/netstatusevent/level"
	"github.com/studease/common/log"
	"github.com/studease/common/rtmp/config"
	rtmpcfg "github.com/studease/common/rtmp/config"
	"github.com/studease/common/rtmp/message"
	"github.com/studease/common/rtmp/message/command"
	CSID "github.com/studease/common/rtmp/message/csid"
	EventType "github.com/studease/common/rtmp/message/eventtype"
	LimitType "github.com/studease/common/rtmp/message/limittype"
	basecfg "github.com/studease/common/utils/config"
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

// LiveHandler provides live broadcast service
type LiveHandler struct {
	srv     *Server
	cfg     *rtmpcfg.Location
	logger  log.ILogger
	factory log.ILoggerFactory
	mtx     sync.RWMutex
	table   map[string]*Application

	connectListener      *events.EventListener
	createStreamListener *events.EventListener
	publishListener      *events.EventListener
	playListener         *events.EventListener
	closeStreamListener  *events.EventListener
	closeListener        *events.EventListener
}

// Init this class
func (me *LiveHandler) Init(srv *Server, cfg *config.Location, logger log.ILogger, factory log.ILoggerFactory) IHandler {
	me.srv = srv
	me.cfg = cfg
	me.logger = logger
	me.factory = factory
	me.table = make(map[string]*Application)
	me.connectListener = events.NewListener(me.onConnect, 0)
	me.createStreamListener = events.NewListener(me.onCreateStream, 0)
	me.publishListener = events.NewListener(me.onPublish, 0)
	me.playListener = events.NewListener(me.onPlay, 0)
	me.closeStreamListener = events.NewListener(me.onCloseStream, 0)
	me.closeListener = events.NewListener(me.onClose, 0)
	return me
}

// ServeRTMP handles a NetConnection
func (me *LiveHandler) ServeRTMP(nc *NetConnection) error {
	nc.AddEventListener(CommandEvent.CONNECT, me.connectListener)
	nc.AddEventListener(CommandEvent.CREATE_STREAM, me.createStreamListener)
	nc.AddEventListener(CommandEvent.CLOSE, me.closeListener)
	return nil
}

func (me *LiveHandler) onConnect(e *CommandEvent.CommandEvent) {
	nc := e.Target.(*NetConnection)
	m := e.Message

	if url := &me.cfg.OnOpen; url.Enable {
		err := nc.sendNotification(url, "connect")
		if err != nil {
			me.logger.Errorf("Failed to send \"connect\" notification: %v", err)
			me.reject(nc, err.Error())
			return
		}
	}

	me.accept(nc)

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

	ns := new(NetStream).Init(nc, me.srv, me.logger, me.factory)
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
	nc := ns.Connection()
	m := e.Message

	stream := me.getStream(nc.AppName, nc.InstName, m.PublishingName)
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

	ns.Sink(stream)
	me.record(stream)

	// Publish to proxy
}

func (me *LiveHandler) onPlay(e *CommandEvent.CommandEvent) {
	ns := e.Target.(*NetStream)
	nc := ns.Connection()

	if url := &me.cfg.OnPlay; url.Enable {
		err := ns.sendNotification(url, "play")
		if err != nil {
			me.logger.Errorf("Failed to send \"play\" notification: %v", err)
			ns.SendStatus(Level.ERROR, Code.NETSTREAM_PLAY_FAILED, err.Error())
			ns.Close()
			return
		}
	}

	stream := me.getStream(nc.AppName, nc.InstName, ns.Name())
	if stream == nil {
		me.logger.Errorf("Failed to get stream")
		ns.SendStatus(Level.ERROR, Code.NETSTREAM_FAILED, "internal error")
		nc.Close()
		return
	}

	ns.Source(stream)
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

func (me *LiveHandler) onClose(e *CommandEvent.CommandEvent) {
	nc := e.Target.(*NetConnection)

	if url := &me.cfg.OnClose; url.Enable {
		err := nc.sendNotification(url, "close")
		if err != nil {
			me.logger.Errorf("Failed to send \"close\" notification: %v", err)
			return
		}
	}
}

func (me *LiveHandler) accept(nc *NetConnection) {
	me.logger.Debugf(4, "Accepting connection: app=%s, inst=%s, id=%s", nc.AppName, nc.InstName, nc.FarID)

	me.mtx.Lock()
	defer me.mtx.Unlock()

	app, ok := me.table[nc.AppName]
	if !ok {
		app = new(Application).Init(nc.AppName, me.logger, me.factory)
		me.table[nc.AppName] = app
	}
}

func (me *LiveHandler) reject(nc *NetConnection, description string) {
	nc.reply(command.ERROR, 1, Level.ERROR, Code.NETCONNECTION_CONNECT_REJECTED, description)
	nc.Close()
}

func (me *LiveHandler) getStream(appName string, instName string, name string) *Stream {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	app, ok := me.table[appName]
	if !ok {
		app = new(Application).Init(appName, me.logger, me.factory)
		me.table[appName] = app
	}

	stream := app.Get(instName, name)

	if atomic.CompareAndSwapUint32(&stream.readyState, STREAM_IDLE, STREAM_PUBLISHING) {
		// Play from proxy
	}

	return stream
}

func (me *LiveHandler) record(stream *Stream) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}
	}()

	for i, cfg := range me.cfg.DVRs {
		_, ok := stream.DVRs[cfg.ID]
		if ok || strings.Contains(cfg.Mode, "off") {
			continue
		}

		if cfg.Mode == "" {
			me.cfg.DVRs[i].Mode = "all"
		}

		rec := dvr.New(cfg.Name, &me.cfg.DVRs[i], me.factory)

		if !strings.Contains(cfg.Mode, "manual") {
			rec.Attach(stream)
		}

		stream.DVRs[cfg.ID] = rec
	}
}

// NewInfoObject creates a rtmp info object
func NewInfoObject(level string, code string, description string) *amf.Value {
	info := amf.NewValue(amf.OBJECT)
	info.Add(amf.NewValue(amf.STRING).Set("level", level))
	info.Add(amf.NewValue(amf.STRING).Set("code", code))
	info.Add(amf.NewValue(amf.STRING).Set("description", description))
	return info
}
