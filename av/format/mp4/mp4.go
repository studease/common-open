package mp4

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// Box types
const (
	TYPE_AVC1 = "avc1"
	TYPE_AVCC = "avcC"
	TYPE_BTRT = "btrt"
	TYPE_DINF = "dinf"
	TYPE_DREF = "dref"
	TYPE_ESDS = "esds"
	TYPE_FTYP = "ftyp"
	TYPE_HDLR = "hdlr"
	TYPE_MDAT = "mdat"
	TYPE_MDHD = "mdhd"
	TYPE_MDIA = "mdia"
	TYPE_MFHD = "mfhd"
	TYPE_MINF = "minf"
	TYPE_MOOF = "moof"
	TYPE_MOOV = "moov"
	TYPE_MP4A = "mp4a"
	TYPE_MVEX = "mvex"
	TYPE_MVHD = "mvhd"
	TYPE_SDTP = "sdtp"
	TYPE_STBL = "stbl"
	TYPE_STCO = "stco"
	TYPE_STSC = "stsc"
	TYPE_STSD = "stsd"
	TYPE_STSZ = "stsz"
	TYPE_STTS = "stts"
	TYPE_TFDT = "tfdt"
	TYPE_TFHD = "tfhd"
	TYPE_TRAF = "traf"
	TYPE_TRAK = "trak"
	TYPE_TRUN = "trun"
	TYPE_TREX = "trex"
	TYPE_TKHD = "tkhd"
	TYPE_VMHD = "vmhd"
	TYPE_SMHD = "smhd"
)

// MP4 is used as the base class of any object about MP4
type MP4 struct {
	format.MediaStream

	logger         log.ILogger
	factory        log.ILoggerFactory
	info           av.Information
	InfoFrame      *av.Packet
	AudioInfoFrame *av.Packet
	VideoInfoFrame *av.Packet
}

// Init this class
func (me *MP4) Init(logger log.ILogger, factory log.ILoggerFactory) *MP4 {
	me.MediaStream.Init()
	me.info.Init()
	me.logger = logger
	me.factory = factory
	return me
}

// NewTrack creates a Track with the given codec, and add it in this MediaStream
func (me *MP4) NewTrack(codec av.Codec) *Track {
	track := new(Track).Init(codec, &me.info, me.logger, me.factory)
	me.AddTrack(track)
	return track
}

// Information returns the associated Information
func (me *MP4) Information() *av.Information {
	return &me.info
}

// Format returns an FMP4 segment with the given arguments
func (me *MP4) Format(pkt *av.Packet) []byte {
	var (
		track av.IMediaTrack
	)

	switch pkt.Type {
	case av.TYPE_AUDIO:
		track = me.AudioTrack()

	case av.TYPE_VIDEO:
		track = me.VideoTrack()

	default:
		panic(fmt.Sprintf("unrecognized packet type 0x%02X", pkt.Type))
	}

	if track == nil {
		me.logger.Debugf(0, "Track not found while formating FMP4 segment: type=%02X", pkt.Type)
		return nil
	}

	return track.(*Track).Format(pkt)
}

// GetInitSegment returns an FMP4 init segment with all tracks inside
func (me *MP4) GetInitSegment() []byte {
	tracks := me.GetTracks()

	boxes := make([][]byte, 0)
	trexs := make([][]byte, 0)

	mvhd := MVHD(me.Information())
	boxes = append(boxes, mvhd)

	for _, track := range tracks {
		trak := TRAK(track.(*Track))
		trex := TREX(track.(*Track))
		boxes = append(boxes, trak)
		trexs = append(trexs, trex)
	}

	mvex := Box(TYPE_MVEX, trexs...)
	boxes = append(boxes, mvex)

	ftyp := FTYP()
	moov := Box(TYPE_MOOV, boxes...)

	return Merge(ftyp, moov)
}
