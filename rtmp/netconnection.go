package rtmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/events"
	CommandEvent "github.com/studease/common/events/commandevent"
	Event "github.com/studease/common/events/event"
	NetStatusEvent "github.com/studease/common/events/netstatusevent"
	"github.com/studease/common/log"
	"github.com/studease/common/rtmp/message"
	"github.com/studease/common/rtmp/message/command"
	CSID "github.com/studease/common/rtmp/message/csid"
	EventType "github.com/studease/common/rtmp/message/eventtype"
	"github.com/studease/common/rtmp/message/support"
	"github.com/studease/common/target"
	basecfg "github.com/studease/common/utils/config"
)

// NetConnection states
const (
	STATE_INITIALIZED = 0x00
	STATE_CONNECTED   = 0x01
	STATE_CLOSING     = 0x02
	STATE_CLOSED      = 0x03
)

var (
	farID     uint32
	pathRe, _ = regexp.Compile("^/([-\\.[:word:]]+)(?:/([-\\.[:word:]]+))?$")
)

// INetStream defines methods to handle rtmp massages.
type INetStream interface {
	process(ck *message.Message) error
	setBufferLength(n uint32)
	Close()
}

// NetConnection creates a two-way connection between a client and a server
type NetConnection struct {
	events.EventDispatcher

	conn              net.Conn
	srv               *Server
	logger            log.ILogger
	factory           log.ILoggerFactory
	mtx               sync.RWMutex
	handshaker        Handshaker
	id                uint32 // Should always be 0
	bufferLength      uint32 // ms
	farAckWindowSize  uint32
	farBandwidth      uint32
	farChunkSize      int32
	farLimitType      uint8
	handlers          map[string]func(*message.CommandMessage) error
	headersOut        map[uint32]*message.Header
	lastAckWindowSize uint32
	message           *message.Message
	messages          map[uint32]*message.Message
	nearAckWindowSize uint32
	neerBandwidth     uint32
	nearChunkSize     int32
	neerLimitType     byte
	responders        map[uint64]*Responder
	state             byte
	streamIndex       uint32
	streams           map[uint32]INetStream
	transactionID     uint64
	readyState        uint32

	Agent             string
	AppName           string
	AudioCodecs       uint64
	AudioSampleAccess string
	BytesIn           uint32
	BytesOut          uint32
	ConnectTime       int64
	FarID             string
	InstName          string
	IP                string
	MsgDropped        uint32
	MsgIn             uint32
	MsgOut            uint32
	MuxerType         string
	NearID            string
	ObjectEncoding    byte
	PageURL           string
	Protocol          string
	ProtocolVersion   string
	ReadAccess        string
	Referrer          string
	Secure            bool
	URL               *url.URL
	VideoCodecs       uint64
	VideoSampleAccess string
	VirtualKey        string
	WriteAccess       string
}

// Init this class
func (me *NetConnection) Init(conn net.Conn, srv *Server, logger log.ILogger, factory log.ILoggerFactory) *NetConnection {
	me.EventDispatcher.Init(logger)
	me.handshaker.Init(conn, logger)
	me.conn = conn
	me.srv = srv
	me.logger = logger
	me.factory = factory
	me.bufferLength = 100
	me.farAckWindowSize = 2500000
	me.farChunkSize = 128
	me.headersOut = make(map[uint32]*message.Header)
	me.lastAckWindowSize = 0
	me.messages = make(map[uint32]*message.Message)
	me.nearChunkSize = 128
	me.responders = make(map[uint64]*Responder)
	me.streamIndex = 0
	me.streams = make(map[uint32]INetStream)
	me.streams[0] = me // Control stream for protocol control messages
	me.transactionID = 0
	me.readyState = STATE_INITIALIZED

	me.FarID = fmt.Sprintf("%d", atomic.AddUint32(&farID, 1))
	me.InstName = "_definst_"
	me.ObjectEncoding = AMF0
	me.ReadAccess = "/"
	me.WriteAccess = "/"
	me.AudioSampleAccess = "/"
	me.VideoSampleAccess = "/"

	me.handlers = map[string]func(*message.CommandMessage) error{
		command.CONNECT:         me.processCommandConnect,
		command.CREATE_STREAM:   me.processCommandCreateStream,
		command.DELETE_STREAM:   me.processCommandDeleteStream,
		command.CLOSE:           me.processCommandClose,
		command.RESULT:          me.processCommandResult,
		command.ERROR:           me.processCommandError,
		command.CHECK_BANDWIDTH: me.processCommandCheckBandwidth,
		command.GET_STATS:       me.processCommandGetStats,
	}

	return me
}

func (me *NetConnection) serve() {
	var (
		b = make([]byte, 14+4096)
	)

	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}

		me.Close()
	}()

	err := me.handshaker.serve()
	if err != nil {
		me.logger.Debugf(4, "Handshaking error: %v", err)
		return
	}

	me.read(b)
}

func (me *NetConnection) read(b []byte) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Errorf("Unexpected error occurred: %v", err)
		}

		me.Close()
	}()

	for {
		n, err := me.conn.Read(b)
		if err != nil {
			me.logger.Debugf(4, "Failed to read: %v", err)
			return
		}

		in := atomic.AddUint32(&me.BytesIn, uint32(n))

		if in-me.lastAckWindowSize >= me.farAckWindowSize {
			err = me.SendAckSequenceNumber()
			if err != nil {
				me.logger.Errorf("Failed to send ack sn: %v", err)
				return
			}

			me.lastAckWindowSize = in
		}

		err = me.parseMessage(b[:n])
		if err != nil {
			me.logger.Errorf("Failed to process message: %v", err)
			return
		}
	}
}

func (me *NetConnection) parseMessage(data []byte) error {
	const (
		sw_fmt byte = iota
		sw_csid_0
		sw_csid_1
		sw_timestamp_0
		sw_timestamp_1
		sw_timestamp_2
		sw_length_0
		sw_length_1
		sw_length_2
		sw_type_id
		sw_stream_id_0
		sw_stream_id_1
		sw_stream_id_2
		sw_stream_id_3
		sw_timestamp_ext0
		sw_timestamp_ext1
		sw_timestamp_ext2
		sw_timestamp_ext3
		sw_data
	)

	size := len(data)

	m := me.message
	if m == nil {
		m = message.New()
		me.message = m
	}

	for i := 0; i < size; i++ {
		ch := uint32(data[i])

		switch me.state {
		case sw_fmt:
			m.FMT = data[i] >> 6
			m.CSID = ch & 0x3F

			if m.Flag == message.FLAG_UNSET {
				if m.FMT == 0 {
					m.Flag = message.FLAG_ABSOLUTE
				} else {
					m.Flag = message.FLAG_DELTA
				}
			}

			switch m.CSID {
			case 0:
				me.state = sw_csid_1
			case 1:
				me.state = sw_csid_0
			default:
				last, ok := me.messages[m.CSID]
				if ok {
					m.Header = last.Header
					if m.FMT == 3 {
						m.Flag = last.Flag
					}
				}

				me.messages[m.CSID] = m

				if m.FMT == 3 {
					if m.Timestamp < 0xFFFFFF {
						me.state = sw_data
					} else {
						me.state = sw_timestamp_ext0
					}
				} else {
					me.state = sw_timestamp_0
				}
			}

		case sw_csid_0:
			m.CSID = ch
			me.state = sw_csid_1

		case sw_csid_1:
			if m.CSID == 0 {
				m.CSID = ch
			} else {
				m.CSID |= ch << 8
			}

			m.CSID += 64

			last, ok := me.messages[m.CSID]
			if ok {
				m.Header = last.Header
				if m.FMT == 3 {
					m.Flag = last.Flag
				}
			}

			me.messages[m.CSID] = m

			if m.FMT == 3 {
				if m.Timestamp < 0xFFFFFF {
					me.state = sw_data
				} else {
					me.state = sw_timestamp_ext0
				}
			} else {
				me.state = sw_timestamp_0
			}

		case sw_timestamp_0:
			m.Timestamp = ch << 16
			me.state = sw_timestamp_1

		case sw_timestamp_1:
			m.Timestamp |= ch << 8
			me.state = sw_timestamp_2

		case sw_timestamp_2:
			m.Timestamp |= ch

			if m.FMT == 2 {
				if m.Timestamp < 0xFFFFFF {
					me.state = sw_data
				} else {
					me.state = sw_timestamp_ext0
				}
			} else {
				me.state = sw_length_0
			}

		case sw_length_0:
			m.Length = ch << 16
			me.state = sw_length_1

		case sw_length_1:
			m.Length |= ch << 8
			me.state = sw_length_2

		case sw_length_2:
			m.Length |= ch
			me.state = sw_type_id

		case sw_type_id:
			m.TypeID = data[i]

			if m.FMT == 1 {
				if m.Timestamp < 0xFFFFFF {
					me.state = sw_data
				} else {
					me.state = sw_timestamp_ext0
				}
			} else {
				me.state = sw_stream_id_0
			}

		case sw_stream_id_0:
			m.StreamID = ch
			me.state = sw_stream_id_1

		case sw_stream_id_1:
			m.StreamID |= ch << 8
			me.state = sw_stream_id_2

		case sw_stream_id_2:
			m.StreamID |= ch << 16
			me.state = sw_stream_id_3

		case sw_stream_id_3:
			m.StreamID |= ch << 24

			if m.Timestamp < 0xFFFFFF {
				me.state = sw_data
			} else {
				me.state = sw_timestamp_ext0
			}

		case sw_timestamp_ext0:
			m.Timestamp = ch << 24
			me.state = sw_timestamp_ext1

		case sw_timestamp_ext1:
			m.Timestamp |= ch << 16
			me.state = sw_timestamp_ext2

		case sw_timestamp_ext2:
			m.Timestamp |= ch << 8
			me.state = sw_timestamp_ext3

		case sw_timestamp_ext3:
			m.Timestamp |= ch
			me.state = sw_data

		case sw_data:
			n := int(m.Length) - m.Buffer.Len()
			if n > size-i {
				n = size - i
			}

			fragmentary := int(me.farChunkSize) - (m.Buffer.Len() % int(me.farChunkSize))
			if n >= fragmentary {
				n = fragmentary
				me.state = sw_fmt
			}

			_, err := m.Buffer.Write(data[i : i+n])
			if err != nil {
				return err
			}

			i += n - 1

			if m.Buffer.Len() == int(m.Length) {
				ns, ok := me.streams[m.StreamID]
				if !ok {
					return fmt.Errorf("message stream %d not found", m.StreamID)
				}

				m.Payload = m.Buffer.Bytes()

				err := ns.process(m)
				if err != nil {
					return err
				}

				m = message.New()
				me.message = m
				me.state = sw_fmt
			}

		default:
			return fmt.Errorf("unrecognized state")
		}
	}

	return nil
}

func (me *NetConnection) process(ck *message.Message) error {
	if ck.TypeID != message.ACK && ck.TypeID != message.USER_CONTROL {
		me.logger.Debugf(4, "onMessage: 0x%02X", ck.TypeID)
	}

	data := ck.Payload

	switch ck.TypeID {
	case message.SET_CHUNK_SIZE:
		me.farChunkSize = int32(binary.BigEndian.Uint32(data) & 0x7FFFFFFF)
		me.logger.Debugf(4, "Set farChunkSize: %d", me.farChunkSize)

	case message.ABORT:
		csid := binary.BigEndian.Uint32(data)
		delete(me.messages, csid)
		me.logger.Debugf(4, "Abort chunk stream: %d", csid)

	case message.ACK:
		sn := binary.BigEndian.Uint32(data)
		me.logger.Debugf(0, "ACK sequence number: %d/%d", sn, me.BytesOut)

	case message.USER_CONTROL:
		m := new(message.UserControlMessage)
		m.Header = ck.Header

		_, err := m.Parse(data)
		if err != nil {
			me.logger.Errorf("Failed to parse user control message: %v", err)
			return err
		}

		return me.processUserControl(m)

	case message.ACK_WINDOW_SIZE:
		me.farAckWindowSize = binary.BigEndian.Uint32(data)
		me.logger.Debugf(4, "Set farAckWindowSize to %d", me.farAckWindowSize)

	case message.BANDWIDTH:
		m := new(message.BandwidthMessage)
		m.Header = ck.Header

		_, err := m.Parse(data)
		if err != nil {
			me.logger.Errorf("Failed to parse bandwidth message: %v", err)
			return err
		}

		return me.processBandwidth(m)

	case message.EDGE:
		// TODO(tonylau): Edge Message

	case message.SHARED_OBJECT_AMF3:
		fallthrough
	case message.SHARED_OBJECT:
		// TODO(tonylau):

	case message.COMMAND_AMF3:
		data = data[1:]
		fallthrough
	case message.COMMAND:
		m := new(message.CommandMessage)
		m.Header = ck.Header

		_, err := m.Parse(data)
		if err != nil {
			me.logger.Errorf("Failed to parse command message: %v", err)
			return err
		}

		return me.processCommand(m)
	}

	return nil
}

func (me *NetConnection) processUserControl(m *message.UserControlMessage) error {
	if m.Event.Type != EventType.PING_REQUEST &&
		m.Event.Type != EventType.PING_RESPONSE &&
		m.Event.Type != EventType.BUFFER_EMPTY &&
		m.Event.Type != EventType.BUFFER_READY {
		me.logger.Debugf(4, "Processing user control message: type=%d", m.Event.Type)
	}

	switch m.Event.Type {
	case EventType.STREAM_BEGIN:
		me.logger.Debugf(4, "Stream Begin: id=%d", m.Event.StreamID)

	case EventType.STREAM_EOF:
		me.logger.Debugf(4, "Stream EOF: id=%d", m.Event.StreamID)

	case EventType.STREAM_DRY:
		me.logger.Debugf(4, "Stream Dry: id=%d", m.Event.StreamID)

	case EventType.SET_BUFFER_LENGTH:
		me.logger.Debugf(4, "Set stream(%d).BufferLength: %dms", m.Event.StreamID, m.Event.BufferLength)

		stream, ok := me.streams[m.Event.StreamID]
		if ok {
			stream.setBufferLength(m.Event.BufferLength)
		}

	case EventType.STREAM_IS_RECORDED:
		me.logger.Debugf(4, "Stream is Recorded: id=%d", m.Event.StreamID)

	case EventType.PING_REQUEST:
		me.logger.Debugf(4, "Ping Request: timestamp=%d", m.Event.Timestamp)
		return me.SendUserControl(EventType.PING_RESPONSE, 0, 0, m.Event.Timestamp)

	case EventType.PING_RESPONSE:
		me.logger.Debugf(4, "Ping Response: timestamp=%d", m.Event.Timestamp)

	case EventType.BUFFER_EMPTY:
		me.logger.Debugf(4, "Stream Buffer Empty: id=%d", m.Event.StreamID)

	case EventType.BUFFER_READY:
		me.logger.Debugf(4, "Stream Buffer Ready: id=%d", m.Event.StreamID)
	}

	return nil
}

func (me *NetConnection) processBandwidth(m *message.BandwidthMessage) error {
	me.logger.Debugf(4, "Set neerBandwidth: ack=%d, limit=%d", m.AckWindowSize, m.LimitType)

	me.neerBandwidth = m.AckWindowSize
	me.neerLimitType = m.LimitType

	return nil
}

func (me *NetConnection) processCommand(m *message.CommandMessage) error {
	me.logger.Debugf(4, "Processing command message: %s", m.CommandName)

	h, ok := me.handlers[m.CommandName]
	if ok {
		return h(m)
	}

	// Should not return error, just ignore
	me.logger.Warnf("No handler found: command=%s, stream=%d", m.CommandName, m.StreamID)

	return nil
}

func (me *NetConnection) processCommandConnect(m *message.CommandMessage) error {
	if atomic.LoadUint32(&me.readyState) == STATE_CONNECTED {
		return fmt.Errorf("already connected")
	}

	v := m.CommandObject.Get("objectEncoding")
	if v != nil && v.Double() == 3 {
		me.ObjectEncoding = AMF3
		me.logger.Warnf("Using ObjectEncoding.AMF3")
	}

	a := m.CommandObject.Get("app")
	u := m.CommandObject.Get("tcUrl")
	if a == nil || u == nil {
		return fmt.Errorf("necessary parameter[s] not found")
	}
	err := me.parseURL(u.String())
	if err != nil {
		return err
	}

	h, _ := me.srv.mux.Handler(me.URL)
	if h != nil {
		h.(IHandler).ServeRTMP(me)
	}

	me.DispatchEvent(CommandEvent.New(CommandEvent.CONNECT, me, m))
	return nil
}

func (me *NetConnection) processCommandCreateStream(m *message.CommandMessage) error {
	me.DispatchEvent(CommandEvent.New(CommandEvent.CREATE_STREAM, me, m))
	return nil
}

func (me *NetConnection) processCommandDeleteStream(m *message.CommandMessage) error {
	id := uint32(m.Arguments.Double())

	me.mtx.Lock()
	stream, ok := me.streams[id]
	delete(me.streams, id)
	me.mtx.Unlock()

	if ok {
		stream.Close()
	}

	return nil
}

func (me *NetConnection) processCommandClose(m *message.CommandMessage) error {
	me.Close()
	return nil
}

func (me *NetConnection) processCommandResult(m *message.CommandMessage) error {
	me.mtx.Lock()
	r, ok := me.responders[m.TransactionID]
	me.mtx.Unlock()

	if ok && r.Result != nil {
		r.Result(m)
	}

	if m.Arguments.Type == amf.OBJECT {
		me.DispatchEvent(NetStatusEvent.New(NetStatusEvent.NET_STATUS, me, &m.Arguments))
	}

	return nil
}

func (me *NetConnection) processCommandError(m *message.CommandMessage) error {
	me.mtx.Lock()
	r, ok := me.responders[m.TransactionID]
	me.mtx.Unlock()

	if ok && r.Status != nil {
		r.Status(m)
	}

	if m.Arguments.Type == amf.OBJECT {
		me.DispatchEvent(NetStatusEvent.New(NetStatusEvent.NET_STATUS, me, &m.Arguments))
	}

	return nil
}

func (me *NetConnection) processCommandCheckBandwidth(m *message.CommandMessage) error {
	return nil
}

func (me *NetConnection) processCommandGetStats(m *message.CommandMessage) error {
	return nil
}

// SetChunkSize is used to notify the peer of a new maximum chunk size
func (me *NetConnection) SetChunkSize(n int32) error {
	var (
		b bytes.Buffer
	)

	amf.AppendInt32(&b, n, false)

	_, err := me.sendBytes(CSID.PROTOCOL_CONTROL, message.SET_CHUNK_SIZE, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	me.nearChunkSize = n
	me.logger.Debugf(4, "Set nearChunkSize: %d", me.nearChunkSize)
	return nil
}

// Abort is used to notify the peer if it is waiting for chunks to complete a message,
// then to discard the partially received message over a chunk stream
func (me *NetConnection) Abort(csid uint32) error {
	var (
		b bytes.Buffer
	)

	amf.AppendUint32(&b, csid, false)

	_, err := me.sendBytes(CSID.COMMAND, message.COMMAND, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// SendAckSequenceNumber sends an acknowledgment to the peer after receiving bytes equal to the window size
func (me *NetConnection) SendAckSequenceNumber() error {
	var (
		b bytes.Buffer
	)

	amf.AppendUint32(&b, me.BytesIn, false)

	_, err := me.sendBytes(CSID.PROTOCOL_CONTROL, message.ACK, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	me.logger.Debugf(0, "Send ack sequence number: %d", me.BytesIn)
	return nil
}

// SendUserControl sends a message to notify the peer about the user control events
func (me *NetConnection) SendUserControl(event uint16, streamID uint32, bufferLength uint32, timestamp uint32) error {
	var (
		b bytes.Buffer
	)

	amf.AppendUint16(&b, event, false)
	if event <= EventType.STREAM_IS_RECORDED {
		amf.AppendUint32(&b, streamID, false)
	}
	if event == EventType.SET_BUFFER_LENGTH {
		amf.AppendUint32(&b, bufferLength, false)
	}
	if event == EventType.PING_REQUEST || event == EventType.PING_RESPONSE {
		amf.AppendUint32(&b, timestamp, false)
	}

	_, err := me.sendBytes(CSID.PROTOCOL_CONTROL, message.USER_CONTROL, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	me.logger.Debugf(4, "Sent user control event: %d", event)
	return nil
}

// SetAckWindowSize sends a message to inform the peer of the window size to use between sending acknowledgments
func (me *NetConnection) SetAckWindowSize(n uint32) error {
	var (
		b bytes.Buffer
	)

	amf.AppendUint32(&b, n, false)

	_, err := me.sendBytes(CSID.PROTOCOL_CONTROL, message.ACK_WINDOW_SIZE, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	me.nearAckWindowSize = n
	me.logger.Debugf(4, "Set nearAckWindowSize: %d", me.nearAckWindowSize)
	return nil
}

// SetPeerBandwidth sends a message to limit the output bandwidth of its peer
func (me *NetConnection) SetPeerBandwidth(n uint32, limitType uint8) error {
	var (
		b bytes.Buffer
	)

	amf.AppendUint32(&b, n, false)
	amf.AppendUint8(&b, limitType)

	_, err := me.sendBytes(CSID.PROTOCOL_CONTROL, message.BANDWIDTH, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	me.farBandwidth = n
	me.farLimitType = limitType
	me.logger.Debugf(4, "Set farBandwidth: ack=%d, limit=%d", me.farBandwidth, me.farLimitType)
	return nil
}

func (me *NetConnection) attach(ns INetStream) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	id := atomic.AddUint32(&me.streamIndex, 1)
	ns.(*NetStream).id = id
	me.streams[id] = ns
}

func (me *NetConnection) getStream(id uint32) INetStream {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	return me.streams[id]
}

func (me *NetConnection) setID(id uint32) {
	// Nothing to do here
}

func (me *NetConnection) setBufferLength(n uint32) {
	me.bufferLength = n
	me.logger.Debugf(4, "Set stream(%d).bufferLength: %d", me.id, me.bufferLength)
}

func (me *NetConnection) sendNotification(url *basecfg.URL, event string) error {
	rawquery := "call=" + event
	rawquery += "&addr=" + me.RemoteAddr()
	rawquery += "&app=" + me.AppName
	rawquery += "&inst=" + me.InstName
	if tmp := me.URL.Query().Encode(); event == "onOpen" && tmp != "" {
		rawquery += "&" + tmp
	}

	res, err := target.Request(url, rawquery)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(res.Status)
	}
	return nil
}

func (me *NetConnection) parseURL(uri string) error {
	var (
		err error
	)

	me.URL, err = url.Parse(uri)
	if err != nil {
		return err
	}

	arr := pathRe.FindStringSubmatch(me.URL.Path)
	if arr == nil {
		return fmt.Errorf("path not matched: %s", me.URL.Path)
	}

	me.AppName = arr[1]
	if arr[2] != "" {
		me.InstName = arr[2]
	}
	return nil
}

// Call a command or method on server
func (me *NetConnection) Call(command string, responder *Responder, args ...*amf.Value) error {
	id := atomic.AddUint64(&me.transactionID, 1)

	if responder != nil {
		me.mtx.Lock()
		me.responders[id] = responder
		me.mtx.Unlock()
	}

	var b bytes.Buffer
	amf.EncodeString(&b, command)
	amf.EncodeDouble(&b, float64(id))
	for _, v := range args {
		amf.Encode(&b, v)
	}

	_, err := me.sendBytes(CSID.COMMAND, message.COMMAND, 0, 0, b.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// Connect creates a two-way connection to an application on server
func (me *NetConnection) Connect(uri string, args ...*amf.Value) error {
	var (
		vals = make([]*amf.Value, 0, 2)
		err  error
	)

	me.logger.Debugf(4, "Connecting to %s...", uri)

	// Always set to 1
	me.transactionID = 0

	// Command Object
	err = me.parseURL(uri)
	if err != nil {
		return err
	}

	v := amf.NewValue(amf.OBJECT)
	v.Add(amf.NewValue(amf.STRING).Set("app", me.AppName))
	v.Add(amf.NewValue(amf.STRING).Set("flashVer", "WIN 32,0,0,114"))
	v.Add(amf.NewValue(amf.STRING).Set("swfUrl", uri))
	v.Add(amf.NewValue(amf.STRING).Set("tcUrl", uri))
	v.Add(amf.NewValue(amf.BOOLEAN).Set("fpad", false))
	v.Add(amf.NewValue(amf.DOUBLE).Set("capabilities", float64(239)))
	v.Add(amf.NewValue(amf.DOUBLE).Set("audioCodecs", float64(support.SND_AAC)))
	v.Add(amf.NewValue(amf.DOUBLE).Set("videoCodecs", float64(support.VID_H264)))
	v.Add(amf.NewValue(amf.DOUBLE).Set("videoFunction", float64(1)))

	vals = append(vals, v)

	// Optional User Arguments
	if len(args) > 0 {
		v = amf.NewValue(amf.OBJECT)
		for _, e := range args {
			v.Add(e)
		}

		vals = append(vals, v)
	}

	return me.Call(command.CONNECT, NewResponder(func(m *message.CommandMessage) {
		atomic.StoreUint32(&me.readyState, STATE_CONNECTED)
	}, nil), vals...)
}

// CreateStream sends a createStream command to server
func (me *NetConnection) CreateStream(r *Responder) error {
	me.logger.Debugf(4, "Creating stream...")
	return me.Call(command.CREATE_STREAM, r, amf.NewValue(amf.NULL))
}

// RemoteAddr returns the remote network address
func (me *NetConnection) RemoteAddr() string {
	return me.conn.RemoteAddr().String()
}

func (me *NetConnection) reply(cmd string, transactionID uint64, level string, code string, description string) error {
	info := NewInfoObject(level, code, description)

	var b bytes.Buffer
	amf.EncodeString(&b, cmd)
	amf.EncodeDouble(&b, float64(transactionID))
	amf.Encode(&b, fmsProperties)
	amf.Encode(&b, info)

	_, err := me.sendBytes(CSID.COMMAND, message.COMMAND, 0, 0, b.Bytes())
	return err
}

func (me *NetConnection) sendBytes(csid uint32, typ byte, timestamp uint32, streamID uint32, data []byte) (int, error) {
	var (
		b bytes.Buffer
		f byte
		i int
		x int
		n = len(data)
	)

	if me.ObjectEncoding == AMF3 {
		switch typ {
		case message.DATA:
			typ = message.DATA_AMF3
		case message.SHARED_OBJECT:
			typ = message.SHARED_OBJECT_AMF3
		case message.COMMAND:
			typ = message.COMMAND_AMF3
		}
	}

	last, ok := me.headersOut[csid]
	if ok {
		if streamID == last.StreamID {
			f = 1
			if typ == last.TypeID && n == int(last.Length) {
				f = 2
				if timestamp == last.Timestamp {
					f = 3
				}
			}
		}
	} else {
		last = new(message.Header)
		me.headersOut[csid] = last
	}

	last.Timestamp = timestamp
	last.Length = uint32(n)
	last.TypeID = typ
	last.StreamID = streamID

	for i < n {
		if csid < 64 {
			b.WriteByte((f << 6) | byte(csid))
		} else if csid < 320 {
			b.WriteByte((f << 6))
			b.WriteByte(byte(csid - 64))
		} else if csid < 65600 {
			tmp := uint16(csid - 64)
			b.WriteByte((f << 6) | 0x01)
			err := binary.Write(&b, binary.LittleEndian, &tmp)
			if err != nil {
				return i, err
			}
		} else {
			return i, fmt.Errorf("CSID \"%d\" out of range", csid)
		}

		// Timestamp
		if f < 3 {
			if timestamp < 0xFFFFFF {
				b.Write([]byte{byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp)})
			} else {
				b.Write([]byte{0xFF, 0xFF, 0xFF})
			}
		}

		// Message Length, Message Type ID
		if f < 2 {
			b.Write([]byte{byte(n >> 16), byte(n >> 8), byte(n)})
			b.WriteByte(typ)
		}

		// Message Stream ID
		if f == 0 {
			binary.Write(&b, binary.LittleEndian, &streamID)
		}

		// Extended Timestamp
		if timestamp >= 0xFFFFFF {
			binary.Write(&b, binary.BigEndian, &timestamp)
		}

		// Payload Data
		j := n - i
		if j > int(me.nearChunkSize) {
			j = int(me.nearChunkSize)
		}

		_, err := b.Write(data[i : i+j])
		if err != nil {
			return i, err
		}

		// Write Chunk
		x, err = me.conn.Write(b.Bytes())
		if err != nil {
			return i, err
		}

		i += j
		atomic.AddUint32(&me.BytesOut, uint32(x))

		f = 3
		b.Reset()
	}

	return i, nil
}

// Close the connection that was opened locally or to the server and dispatches a netStatus event with a code property of NetConnection.Connect.Closed
func (me *NetConnection) Close() {
	switch atomic.LoadUint32(&me.readyState) {
	case STATE_CONNECTED:
		atomic.StoreUint32(&me.readyState, STATE_CLOSING)

		for _, stream := range me.streams {
			if stream != me {
				stream.Close()
			}
		}
		me.conn.Close()
		me.DispatchEvent(Event.New(Event.CLOSE, me))
		fallthrough
	case STATE_INITIALIZED:
		atomic.StoreUint32(&me.readyState, STATE_CLOSED)
		me.conn.Close()
	}
}
