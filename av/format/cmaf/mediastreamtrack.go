package cmaf

import (
	"os"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// MediaChunk describes a LL-CMAF chunk.
type MediaChunk struct {
	Minor       int
	URI         string
	Timestamp   uint32
	Duration    uint32
	Offset      uint32
	Size        uint32
	Independent bool
	Frames      []*av.Packet
}

// Init this class.
func (me *MediaChunk) Init(index int) *MediaChunk {
	me.Minor = index
	me.Frames = make([]*av.Packet, 0)
	return me
}

// MediaSegment describes a LL-CMAF segment.
type MediaSegment struct {
	Major       int
	URI         string
	Timestamp   uint32
	Duration    uint32
	Size        uint32
	Independent bool
	Chunks      []*MediaChunk // completed chunks
	Chunk       *MediaChunk   // uncompleted chunk
}

// Init this class.
func (me *MediaSegment) Init(index int) *MediaSegment {
	me.Major = index
	me.Chunks = make([]*MediaChunk, 0)
	return me
}

// MediaStreamTrack inherits from MediaStreamTrack, holds properties which describes a LL-CMAF track.
type MediaStreamTrack struct {
	format.MediaStreamTrack

	logger      log.ILogger
	InitSegment *MediaChunk     // init segment of this single track
	Segments    []*MediaSegment // completed segments
	Segment     *MediaSegment   // uncompleted segment with completed chunks
	File        *os.File        // for uncompleted segment
}

// Init this class.
func (me *MediaStreamTrack) Init(kind string, source av.IMediaStreamTrackSource, logger log.ILogger) *MediaStreamTrack {
	me.MediaStreamTrack.Init(kind, source, logger)
	me.logger = logger
	me.Segments = make([]*MediaSegment, 0)
	return me
}

// Stop detaches from the source.
func (me *MediaStreamTrack) Stop() {
	me.MediaStreamTrack.Stop()
	me.InitSegment = nil
	me.Segments = nil
	me.Segment = nil
	if me.File != nil {
		me.File.Close()
		me.File = nil
	}
}
