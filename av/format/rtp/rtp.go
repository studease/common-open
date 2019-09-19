package rtp

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// Static constants
const (
	Version   byte  = 2
	MTU       int   = 1500
	H264_FREQ int64 = 90000
)

// RTP is used as the base class of any object about RTP
type RTP struct {
	format.MediaStream

	logger         log.ILogger
	factory        log.ILoggerFactory
	info           av.Information
	InfoFrame      *av.Packet
	AudioInfoFrame *av.Packet
	VideoInfoFrame *av.Packet
}

// Init this class
func (me *RTP) Init(logger log.ILogger, factory log.ILoggerFactory) *RTP {
	me.MediaStream.Init()
	me.info.Init()
	me.logger = logger
	me.factory = factory
	return me
}

// NewTrack creates a Track with the given codec, and add it in this MediaStream
func (me *RTP) NewTrack(codec av.Codec) *Track {
	track := new(Track).Init(codec, &me.info, me.logger, me.factory)
	me.AddTrack(track)
	return track
}

// Information returns the associated Information
func (me *RTP) Information() *av.Information {
	return &me.info
}

// Format returns a sequence of RTP packets with the given arguments
func (me *RTP) Format(pkt *av.Packet) []*Packet {
	var (
		track av.IMediaTrack
	)

	switch pkt.Type {
	case av.TYPE_AUDIO:
		track = me.AudioTrack()

	case av.TYPE_VIDEO:
		track = me.VideoTrack()

	default:
		panic(fmt.Sprintf("unrecognized packet type %02X", pkt.Type))
	}

	if track == nil {
		me.logger.Debugf(0, "Track not found while formating RTP packets: type=%02X", pkt.Type)
		return []*Packet{}
	}

	return track.(*Track).Format(pkt)
}
