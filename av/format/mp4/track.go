package mp4

import (
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/codec/aac"
	"github.com/studease/common/av/codec/avc"
	"github.com/studease/common/av/format"
	"github.com/studease/common/log"
)

// Track inherit from MediaTrack, provides methods to format MP4 fragments
type Track struct {
	format.MediaTrack

	logger         log.ILogger
	SequenceNumber uint32
}

// Init this class
func (me *Track) Init(codec av.Codec, info *av.Information, logger log.ILogger, factory log.ILoggerFactory) *Track {
	me.MediaTrack.Init(codec, info, logger, factory)
	me.logger = logger
	me.SequenceNumber = 0
	return me
}

// Format returns an FMP4 segment with the given arguments
func (me *Track) Format(pkt *av.Packet) []byte {
	switch pkt.Codec {
	case codec.AAC:
		switch pkt.DataType {
		case aac.SPECIFIC_CONFIG:
			return me.getInitSegment()
		case aac.RAW_FRAME_DATA:
			return me.getAudioSegment()
		default:
			panic(fmt.Sprintf("unrecognized AAC type 0x%02X", pkt.DataType))
		}

	case codec.AVC:
		switch pkt.DataType {
		case avc.SEQUENCE_HEADER:
			return me.getInitSegment()
		case avc.NALU:
			fallthrough
		case avc.END_OF_SEQUENCE:
			return me.getVideoSegment()
		default:
			panic(fmt.Sprintf("unrecognized AVC type 0x%02X", pkt.DataType))
		}

	default:
		panic(fmt.Sprintf("unrecognized codec 0x%02X", pkt.Codec))
	}
}

func (me *Track) getInitSegment() []byte {
	ftyp := FTYP()
	moov := MOOV(me)
	return Merge(ftyp, moov)
}

func (me *Track) getAudioSegment() []byte {
	var (
		moof, mdat []byte
	)

	info := me.Information()
	ctx := me.Context().Basic()

	ctx.CTS = 0
	ctx.DTS -= info.TimeBase

	delta := ctx.DTS - ctx.ExpectedDts
	ctx.DTS = ctx.ExpectedDts
	ctx.PTS = ctx.CTS + ctx.DTS
	ctx.Duration = info.RefSampleDuration + delta
	ctx.ExpectedDts = ctx.DTS + ctx.Duration

	me.SequenceNumber++

	moof = MOOF(me)
	mdat = MDAT(ctx.Data)

	return Merge(moof, mdat)
}

func (me *Track) getVideoSegment() []byte {
	var (
		moof, mdat []byte
	)

	info := me.Information()
	ctx := me.Context().Basic()

	ctx.DTS -= info.TimeBase

	delta := ctx.DTS - ctx.ExpectedDts
	ctx.DTS = ctx.ExpectedDts
	ctx.PTS = ctx.CTS + ctx.DTS
	ctx.Duration = info.RefSampleDuration + delta
	ctx.ExpectedDts = ctx.DTS + ctx.Duration
	if ctx.Keyframe {
		ctx.Flags.SampleDependsOn = 2
		ctx.Flags.SampleIsDependedOn = 1
	} else {
		ctx.Flags.SampleDependsOn = 1
		ctx.Flags.SampleIsDependedOn = 0
	}

	me.SequenceNumber++

	moof = MOOF(me)
	mdat = MDAT(ctx.Data)

	return Merge(moof, mdat)
}
