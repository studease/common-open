package av

import (
	"github.com/studease/common/events"
	"github.com/studease/common/log"
)

// Media kinds
const (
	KIND_AUDIO = "audio"
	KIND_VIDEO = "video"
)

// Track ready states
const (
	TRACK_LIVE  = "live"
	TRACK_ENDED = "ended"
)

// ISegmableStream defines the basic segmable stream
type ISegmableStream interface {
	ISeekableStream

	AddSourceBuffer(kind string, codec Codec) ISourceBuffer
	RemoveSourceBuffer(b ISourceBuffer)
}

// ISourceBuffer represents a chunk of media
type ISourceBuffer interface {
	ISeekableStream

	Append(p *Packet)
	Write(name string, timestamp uint32, data []byte) (interface{}, error)
	Timestamp() uint32
	Bytes() []byte
	Len() int
}

// ISeekableStream defines the basic seekable stream
type ISeekableStream interface {
	AddPoint(timestamp uint32, value interface{})
	GetValue(timestamp uint32, n int) []interface{}
	GetLastN(n int) []interface{}
	Clear()
}

// IReadableStream defines the basic readable stream
type IReadableStream interface {
	events.IEventDispatcher
	IMediaStream

	Information() *Information
	AppName() string
	InstName() string
	Name() string
	Parameters() string

	SetDataFrame(key string, p *Packet)
	ClearDataFrame(key string)
	GetDataFrame(key string) *Packet

	InfoFrame() *Packet
	SetAudioInfoFrame(p *Packet)
	SetVideoInfoFrame(p *Packet)
	GetAudioInfoFrame() *Packet
	GetVideoInfoFrame() *Packet
}

// ISinkableStream defines the basic sinkable stream
type ISinkableStream interface {
	Sink(pkt *Packet) error
}

// IMediaStream defines methods to manage tracks
type IMediaStream interface {
	AddTrack(track IMediaTrack)
	RemoveTrack(track IMediaTrack)
	AudioTrack() IMediaTrack
	VideoTrack() IMediaTrack
	GetTracks() []IMediaTrack
	GetTrackByID(id int) IMediaTrack
	Close() error
}

// IMediaTrack represents a single media track within a stream
type IMediaTrack interface {
	SetID(id int)
	ID() int
	Information() *Information
	Context() IMediaContext
	Kind() string
	ReadyState() string
	Close() error
}

// IMediaContext defines methods to parse Packet
type IMediaContext interface {
	Init(info *Information, logger log.ILogger) IMediaContext
	Information() *Information
	Basic() *Context
	Codec() Codec
	Parse(p *Packet) error
}
