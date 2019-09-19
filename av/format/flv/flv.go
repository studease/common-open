package flv

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// Tag types
const (
	TYPE_UNKNOWN byte = 0x00
	TYPE_AUDIO   byte = 0x08
	TYPE_VIDEO   byte = 0x09
	TYPE_DATA    byte = 0x12
)

// Audio codecs
const (
	LINEAR_PCM_PLATFORM_ENDIAN   byte = 0x00
	ADPCM                        byte = 0x10
	MP3                          byte = 0x20
	LINEAR_PCM_LITTLE_ENDIAN     byte = 0x30
	NELLYMOSER_16_kHz_MONO       byte = 0x40
	NELLYMOSER_8_kHz_MONO        byte = 0x50
	NELLYMOSER                   byte = 0x60
	G_711_A_LAW_LOGARITHMIC_PCM  byte = 0x70
	G_711_MU_LAW_LOGARITHMIC_PCM byte = 0x80
	RESERVED                     byte = 0x90
	AAC                          byte = 0xA0
	SPEEX                        byte = 0xB0
	MP3_8_kHz                    byte = 0xE0
	DEVICE_SPECIFIC_SOUND        byte = 0xF0
)

// Video codecs
const (
	JPEG           byte = 0x01
	H263           byte = 0x02
	SCREEN_VIDEO   byte = 0x03
	VP6            byte = 0x04
	VP6_ALPHA      byte = 0x05
	SCREEN_VIDEO_2 byte = 0x06
	AVC            byte = 0x07
)

// Static variables, should not change
var (
	types = map[av.Type]byte{
		av.TYPE_AUDIO: TYPE_AUDIO,
		av.TYPE_VIDEO: TYPE_VIDEO,
		av.TYPE_DATA:  TYPE_DATA,
	}

	Header = []byte{
		'F', 'L', 'V',
		0x01,
		0x05,
		0x00, 0x00, 0x00, 0x09,
		0x00, 0x00, 0x00, 0x00,
	}

	Footer = []byte{
		0x17, 0x02, 0x00, 0x00, 0x00,
	}
)

// FLV is used as the base class of any object about FLV
type FLV struct {
	format.MediaStream

	logger         log.ILogger
	factory        log.ILoggerFactory
	info           av.Information
	InfoFrame      *av.Packet
	AudioInfoFrame *av.Packet
	VideoInfoFrame *av.Packet
}

// Init this class
func (me *FLV) Init(logger log.ILogger, factory log.ILoggerFactory) *FLV {
	me.MediaStream.Init()
	me.info.Init()
	me.logger = logger
	me.factory = factory
	return me
}

// NewTrack creates a Track with the given codec, and add it in this MediaStream
func (me *FLV) NewTrack(codec av.Codec) *Track {
	track := new(Track).Init(codec, &me.info, me.logger, me.factory)
	me.AddTrack(track)
	return track
}

// Information returns the associated Information
func (me *FLV) Information() *av.Information {
	return &me.info
}

// Format returns an FLV tag with the given arguments
func (me *FLV) Format(typ av.Type, timestamp uint32, data []byte) []byte {
	var (
		track av.IMediaTrack
	)

	switch typ {
	case av.TYPE_AUDIO:
		track = me.AudioTrack()

	case av.TYPE_VIDEO:
		track = me.VideoTrack()

	case av.TYPE_DATA:
		return Tag(TYPE_DATA, timestamp, data)

	default:
		panic(fmt.Sprintf("unrecognized packet type %02X", typ))
	}

	if track == nil {
		me.logger.Debugf(0, "Track not found while formating FLV tag: type=%02X", typ)
		return nil
	}

	return track.(*Track).Format(timestamp, data)
}
