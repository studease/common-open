package format

import (
	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/log"
)

// MediaTrack is used as the base class for the creation of MediaTrack objects
type MediaTrack struct {
	logger     log.ILogger
	info       *av.Information
	codec      av.Codec
	id         int
	readyState string
	Context    av.IMediaContext
}

// Init this class
func (me *MediaTrack) Init(cc av.Codec, info *av.Information, logger log.ILogger, factory log.ILoggerFactory) *MediaTrack {
	me.codec = cc
	me.info = info
	me.logger = logger
	me.readyState = av.TRACK_LIVE
	me.Context = codec.New(cc, info, factory)
	return me
}

// SetID sets ID of this track
func (me *MediaTrack) SetID(id int) {
	me.id = id
}

// ID returns ID of this track
func (me *MediaTrack) ID() int {
	return me.id
}

// Information returns the associated Information
func (me *MediaTrack) Information() *av.Information {
	return me.info
}

// Kind returns kind of this track
func (me *MediaTrack) Kind() string {
	return av.KIND_AUDIO
}

// ReadyState returns ready state of this track
func (me *MediaTrack) ReadyState() string {
	return me.readyState
}

// Close this track, set state to ended
func (me *MediaTrack) Close() error {
	me.readyState = av.TRACK_ENDED
	return nil
}
