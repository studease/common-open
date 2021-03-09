package av

import (
	"strings"
	"time"

	"github.com/studease/common/events"
	"github.com/studease/common/log"
)

// Packet kinds.
const (
	KindAudio  = "audio"
	KindVideo  = "video"
	KindScript = "script"
)

// Muxer modes.
const (
	ModeNone        uint32 = 0x00
	ModeVideo       uint32 = 0x01
	ModeKeyframe    uint32 = 0x03
	ModeAudio       uint32 = 0x04
	ModeAll         uint32 = 0x05
	ModeInterleaved uint32 = 0x10
	ModeAdvanced    uint32 = 0x20
	ModeManual      uint32 = 0x40
	ModeOff         uint32 = 0x80
)

var (
	// UTC location
	UTC, _ = time.LoadLocation("UTC")
	modes  = map[string]uint32{
		"keyframe":    ModeKeyframe,
		"video":       ModeVideo,
		"audio":       ModeAudio,
		"all":         ModeAll,
		"interleaved": ModeInterleaved,
		"advanced":    ModeAdvanced,
		"manual":      ModeManual,
		"off":         ModeOff,
	}
)

// Mode parses a readable string into muxer mode value.
func Mode(s string, sep string) uint32 {
	var (
		mode = ModeNone
	)

	arr := strings.Split(s, sep)
	for _, v := range arr {
		n, ok := modes[v]
		if ok {
			mode |= n
		}
	}
	return mode
}

// Rational is used to define rational numbers.
type Rational struct {
	Num float64 // Numerator
	Den float64 // Denominator
}

// Init this class.
func (me *Rational) Init(num float64, den float64) *Rational {
	me.Num = num
	me.Den = den
	return me
}

// Information represents the details of MediaStream.
type Information struct {
	StartTime     time.Time
	MimeType      string
	Codecs        []string
	Timescale     uint32
	TimeBase      uint32
	Timestamp     uint32
	Duration      uint32
	Size          int64
	Width         uint32
	Height        uint32
	CodecWidth    uint32
	CodecHeight   uint32
	AudioDataRate uint32
	VideoDataRate uint32
	BitRate       uint32
	FrameRate     Rational
	SampleRate    uint32
	SampleSize    uint32
	Channels      uint32
}

// Init this class.
func (me *Information) Init() *Information {
	me.Timescale = 1000
	me.FrameRate.Init(30, 1)
	return me
}

// Packet carries media data of MediaStreamTrack.
type Packet struct {
	Kind       string
	Codec      string // "AVC", "AAC", etc.
	Length     uint32
	Timestamp  uint32
	StreamID   uint32
	Payload    []byte
	Position   uint32
	properties map[string]interface{}
}

// Init this class.
func (me *Packet) Init() *Packet {
	me.properties = make(map[string]interface{}, 0)
	return me
}

// Left returns the length of unused payload.
func (me *Packet) Left() int {
	return len(me.Payload) - int(me.Position)
}

// Set sets a user-defined key-value pair.
func (me *Packet) Set(key string, value interface{}) {
	me.properties[key] = value
}

// Get returns the user-defined value by the key.
func (me *Packet) Get(key string) interface{} {
	return me.properties[key]
}

// Extends copies all of the properties.
func (me *Packet) Extends(pkt *Packet) {
	for key := range pkt.properties {
		me.properties[key] = pkt.properties[key]
	}
}

// Context carries information of IMediaStreamTrackSource.
type Context struct {
	MimeType          string
	Codec             string
	RefSampleDuration uint32
	Flags             struct {
		IsLeading           byte
		SampleDependsOn     byte
		SampleIsDependedOn  byte
		SampleHasRedundancy byte
		IsNonSync           byte
	}
}

// IMediaStreamTrackSource is used for parsing a specific codec.
type IMediaStreamTrackSource interface {
	events.IEventDispatcher

	Init(info *Information, logger log.ILogger) IMediaStreamTrackSource
	Kind() string
	Context() *Context
	SetInfoFrame(pkt *Packet)
	GetInfoFrame() *Packet
	Sink(pkt *Packet)
	Parse(pkt *Packet) error
}

// IMediaStreamTrack represents a single media track within a stream.
type IMediaStreamTrack interface {
	ID() int
	Kind() string
	Source() IMediaStreamTrackSource
	Stop()
	Clone() IMediaStreamTrack
}

// IMediaStream represents a stream of media content.
type IMediaStream interface {
	events.IEventDispatcher

	AddTrack(track IMediaStreamTrack)
	RemoveTrack(track IMediaStreamTrack)
	GetTrackByID(id int) IMediaStreamTrack
	GetTracks() []IMediaStreamTrack
	GetAudioTracks() []IMediaStreamTrack
	GetVideoTracks() []IMediaStreamTrack
	Attached(source IMediaStreamTrackSource) IMediaStreamTrack
	Information() *Information
	SetDataFrame(key string, pkt *Packet)
	GetDataFrame(key string) *Packet
	ClearDataFrame(key string)
	Close()
}

// IDemuxer parses buffer as this format of IMediaStream.
type IDemuxer interface {
	IMediaStream

	Append(data []byte)
	Reset()
}

// IRemuxer generates buffer in this format of IMediaStream.
type IRemuxer interface {
	IMediaStream

	Init(mode uint32, logger log.ILogger) IRemuxer
	Source(ms IMediaStream)
}

// MediaRecorderConstraints dictionary is used to describe a set of capabilities.
type MediaRecorderConstraints struct {
	Mode        uint32
	Directory   string
	FileName    string
	Unique      bool
	Append      bool
	Chunks      int
	Segments    int
	MaxDuration uint32
	MaxSize     int64
	MaxFrames   int64
}

// IMediaRecorder records a specified IMediaStream.
type IMediaRecorder interface {
	events.IEventDispatcher

	Init(constraints *MediaRecorderConstraints, logger log.ILogger) IMediaRecorder
	Source(ms IMediaStream)
	Start()
	Pause()
	Resume()
	Stop()
	ReadyState() uint32
}

// IToken provides basic operations on a token.
type IToken interface {
	Put(key string, value interface{})
	Get(key string) interface{}
	Del(key string)
	Update(expire int64) (string, error)
	Parse(token string) error
	String() string
}
