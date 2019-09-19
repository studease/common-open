package flv

import (
	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// Track inherit from MediaTrack, provides methods to format MP4 fragments
type Track struct {
	format.MediaTrack

	logger log.ILogger
}

// Init this class
func (me *Track) Init(codec av.Codec, info *av.Information, logger log.ILogger, factory log.ILoggerFactory) *Track {
	me.MediaTrack.Init(codec, info, logger, factory)
	me.logger = logger
	return me
}

// Format returns an FLV tag with the given arguments
func (me *Track) Format(timestamp uint32, data []byte) []byte {
	typ := TYPE_AUDIO

	if me.Kind() == av.KIND_VIDEO {
		typ = TYPE_VIDEO
	}

	return Tag(typ, timestamp, data)
}
