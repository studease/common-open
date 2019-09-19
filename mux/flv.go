package mux

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format/flv"
	"github.com/studease/common/av/utils/amf"
	"github.com/studease/common/events"
	Event "github.com/studease/common/events/event"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
)

func init() {
	Register(TYPE_FLV, FLV{})
}

// FLV MUX
type FLV struct {
	events.EventDispatcher
	flv.FLV

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
	state       uint8
	flags       uint8
	header      uint32
	backPointer uint32
	packet      av.Packet
	buffer      bytes.Buffer
}

// Init this class
func (me *FLV) Init(mode uint32, logger log.ILogger, factory log.ILoggerFactory) IMuxer {
	me.EventDispatcher.Init(logger)
	me.FLV.Init(logger, factory)
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
func (me *FLV) Attach(stream av.IReadableStream) {
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

	pkt := (&av.Packet{}).Clone(flv.Header)
	me.DispatchEvent(MediaEvent.New(MediaEvent.DATA, me, pkt))
	me.Size += int64(pkt.Length)

	if me.InfoFrame != nil {
		me.onMetaData(me.InfoFrame.Value)
		me.forward(me.InfoFrame)
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

func (me *FLV) onDataPacket(e *MediaEvent.MediaEvent) {
	pkt := e.Packet

	if pkt.Handler == "@setDataFrame" {
		if pkt.Key == "onMetaData" {
			me.InfoFrame = pkt
			me.onMetaData(pkt.Value)
		}

		me.forward(pkt)
	}
}

func (me *FLV) onAudioPacket(e *MediaEvent.MediaEvent) {
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
	}

	me.forward(pkt)
}

func (me *FLV) onVideoPacket(e *MediaEvent.MediaEvent) {
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
	}

	me.forward(pkt)
}

// Append the data for demuxing
func (me *FLV) Append(data []byte) (int, error) {
	const (
		sw_f uint8 = iota
		sw_l
		sw_v
		sw_version
		sw_flags
		sw_header_0
		sw_header_1
		sw_header_2
		sw_header_3
		sw_back_pointer_0
		sw_back_pointer_1
		sw_back_pointer_2
		sw_back_pointer_3
		sw_type
		sw_length_0
		sw_length_1
		sw_length_2
		sw_timestamp_0
		sw_timestamp_1
		sw_timestamp_2
		sw_timestamp_3
		sw_stream_id_0
		sw_stream_id_1
		sw_stream_id_2
		sw_data
	)

	var (
		size = len(data)
		ch   uint32
	)

	for i := 0; i < size; i++ {
		ch = uint32(data[i])

		switch me.state {
		case sw_f:
			if ch != 0x46 {
				return i, fmt.Errorf("not 'F'")
			}

			me.state = sw_l

		case sw_l:
			if ch != 0x4C {
				return i, fmt.Errorf("not 'L'")
			}

			me.state = sw_v

		case sw_v:
			if ch != 0x56 {
				return i, fmt.Errorf("not 'V'")
			}

			me.state = sw_version

		case sw_version:
			if ch != 0x01 {
				return i, fmt.Errorf("bad version \"%02X\"", ch)
			}

			me.state = sw_flags

		case sw_flags:
			me.flags = uint8(ch)
			me.state = sw_header_0

		case sw_header_0:
			me.header = ch << 24
			me.state = sw_header_1

		case sw_header_1:
			me.header |= ch << 16
			me.state = sw_header_1

		case sw_header_2:
			me.header |= ch << 8
			me.state = sw_header_1

		case sw_header_3:
			me.header |= ch
			me.state = sw_back_pointer_0

		case sw_back_pointer_0:
			me.backPointer = ch << 24
			me.state = sw_back_pointer_1

		case sw_back_pointer_1:
			me.backPointer |= ch << 16
			me.state = sw_back_pointer_2

		case sw_back_pointer_2:
			me.backPointer |= ch << 8
			me.state = sw_back_pointer_3

		case sw_back_pointer_3:
			me.backPointer |= ch
			me.state = sw_type

		case sw_type:
			switch data[i] {
			case flv.TYPE_AUDIO:
				me.packet.Type = av.TYPE_AUDIO
			case flv.TYPE_VIDEO:
				me.packet.Type = av.TYPE_VIDEO
			case flv.TYPE_DATA:
				me.packet.Type = av.TYPE_DATA
			default:
				return i, fmt.Errorf("unrecognized tag type \"%02X\"", data[i])
			}

			me.state = sw_length_0

		case sw_length_0:
			me.packet.Length = ch << 16
			me.state = sw_length_1

		case sw_length_1:
			me.packet.Length |= ch << 8
			me.state = sw_length_2

		case sw_length_2:
			me.packet.Length |= ch
			me.state = sw_timestamp_0

		case sw_timestamp_0:
			me.packet.Timestamp = ch << 16
			me.state = sw_timestamp_1

		case sw_timestamp_1:
			me.packet.Timestamp |= ch << 8
			me.state = sw_timestamp_2

		case sw_timestamp_2:
			me.packet.Timestamp |= ch
			me.state = sw_timestamp_3

		case sw_timestamp_3:
			me.packet.Timestamp |= ch << 24
			me.state = sw_stream_id_0

		case sw_stream_id_0:
			me.packet.StreamID = ch << 16
			me.state = sw_stream_id_1

		case sw_stream_id_1:
			me.packet.StreamID |= ch << 8
			me.state = sw_stream_id_2

		case sw_stream_id_2:
			me.packet.StreamID |= ch
			me.state = sw_data

		case sw_data:
			n := int(me.packet.Length) - me.buffer.Len()
			if n > size-i {
				n = size - i
			}

			_, err := me.buffer.Write(data[i : i+n])
			if err != nil {
				return i, err
			}

			i += n - 1

			if me.buffer.Len() == int(me.packet.Length) {
				me.packet.Payload = me.buffer.Bytes()

				switch me.packet.Type {
				case av.TYPE_AUDIO:
					tmp := me.packet.Payload[0]

					switch tmp & 0xF0 {
					case flv.AAC:
						me.packet.Codec = codec.AAC
					default:
						return i, fmt.Errorf("unsupported audio codec \"%02X\"", tmp&0xF0)
					}

					me.packet.SampleRate = (tmp >> 2) & 0x03
					me.packet.SampleSize = (tmp >> 1) & 0x01
					me.packet.SampleType = tmp & 0x01
					me.packet.DataType = me.packet.Payload[1]

					me.onAudioPacket(MediaEvent.New(MediaEvent.AUDIO, me, &me.packet))

				case av.TYPE_VIDEO:
					tmp := me.packet.Payload[0]

					switch tmp & 0x0F {
					case flv.AVC:
						me.packet.Codec = codec.AVC
					default:
						return i, fmt.Errorf("unsupported video codec \"%02X\"", tmp&0xF0)
					}

					me.packet.FrameType = tmp >> 4
					me.packet.DataType = me.packet.Payload[1]

					if me.packet.DataType == avc.END_OF_SEQUENCE {
						me.Close()
						break
					}

					me.onVideoPacket(MediaEvent.New(MediaEvent.VIDEO, me, &me.packet))

				case av.TYPE_DATA:
					me.packet.Handler = "@setDataFrame"

					v := amf.NewValue(amf.STRING)
					j := 0

					n, err := amf.Decode(v, me.packet.Payload[j:])
					if err != nil {
						return i, err
					}

					me.packet.Key = v.String()
					j += n

					v.Init(amf.OBJECT)

					n, err = amf.Decode(v, me.packet.Payload[j:])
					if err != nil {
						return i, err
					}

					me.packet.Value = v
					j += n

					me.onDataPacket(MediaEvent.New(MediaEvent.DATA, me, &me.packet))

				default:
					me.logger.Errorf("Unrecognized tag type \"%02X\"", me.packet.Type)
				}

				me.buffer.Reset()
				me.state = sw_back_pointer_0
			}

		default:
			return i, fmt.Errorf("unrecognized state")
		}
	}

	return size, nil
}

func (me *FLV) forward(raw *av.Packet) {
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

	tag := me.Format(raw.Type, me.Duration+raw.Timestamp, raw.Payload)
	pkt := raw.Clone(tag)

	me.DispatchEvent(MediaEvent.New(event, me, pkt))
	me.Duration += raw.Timestamp
	me.Size += int64(pkt.Length)
	me.Frames++
}

func (me *FLV) onMetaData(o *amf.Value) {
	defer func() {
		if err := recover(); err != nil {
			me.logger.Debugf(3, "Failed to handle onMetaData: %v", err)
		}
	}()

	onMetaData(me.Information(), o)
}

func (me *FLV) onClose(e *Event.Event) {
	me.Close()
}

// Stream returns the attached IReadableStream
func (me *FLV) Stream() av.IReadableStream {
	return me.stream
}

// Mode returns the mode of this muxer
func (me *FLV) Mode() uint32 {
	return me.mode
}

// ReadyState returns the readyState of this muxer
func (me *FLV) ReadyState() uint32 {
	return atomic.LoadUint32(&me.readyState)
}

// Close this muxer
func (me *FLV) Close() {
	switch atomic.LoadUint32(&me.readyState) {
	case STATE_DETECTING:
		fallthrough
	case STATE_ALIVE:
		if (me.mode & MODE_VIDEO) != 0 {
			pkt := (&av.Packet{
				Type:      av.TYPE_VIDEO,
				Timestamp: 0,
				FrameType: av.KEYFRAME,
				DataType:  avc.END_OF_SEQUENCE,
			}).Clone(flv.Footer)
			me.DispatchEvent(MediaEvent.New(MediaEvent.DATA, me, pkt))
			me.Size += int64(pkt.Length)
			me.Frames++
		}

		atomic.StoreUint32(&me.readyState, STATE_CLOSING)

		me.stream.RemoveEventListener(MediaEvent.DATA, me.dataListener)
		me.stream.RemoveEventListener(MediaEvent.AUDIO, me.audioListener)
		me.stream.RemoveEventListener(MediaEvent.VIDEO, me.videoListener)
		me.stream.RemoveEventListener(Event.CLOSE, me.closeListener)
		me.DispatchEvent(Event.New(Event.CLOSE, me))

		me.FLV.Close()
		me.stream = nil
		atomic.StoreUint32(&me.readyState, STATE_CLOSED)
		me.Duration = 0
		me.Size = 0
		me.Frames = 0
	}
}
