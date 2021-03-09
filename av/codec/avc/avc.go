package avc

import (
	"encoding/binary"
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/events"
	MediaEvent "github.com/studease/common/events/mediaevent"
	"github.com/studease/common/log"
)

// Data types.
const (
	SEQUENCE_HEADER = 0x00
	NALU            = 0x01
	END_OF_SEQUENCE = 0x02
)

// NAL types.
const (
	NAL_SLICE           = 1
	NAL_DPA             = 2
	NAL_DPB             = 3
	NAL_DPC             = 4
	NAL_IDR_SLICE       = 5
	NAL_SEI             = 6
	NAL_SPS             = 7
	NAL_PPS             = 8
	NAL_AUD             = 9
	NAL_END_SEQUENCE    = 10
	NAL_END_STREAM      = 11
	NAL_FILLER_DATA     = 12
	NAL_SPS_EXT         = 13
	NAL_AUXILIARY_SLICE = 19
	NAL_FF_IGNORE       = 0xFF0F001
)

func init() {
	codec.Register("AVC", AVC{})
}

// AVC IMediaStreamTrackSource.
type AVC struct {
	events.EventDispatcher

	logger    log.ILogger
	info      *av.Information
	infoframe *av.Packet
	ctx       av.Context

	// Decoder Configuration Record
	AVCC                 []byte
	ConfigurationVersion byte
	ProfileIndication    byte
	ProfileCompatibility byte
	LevelIndication      byte
	NalLengthSize        uint32 // length_size_minus1 + 1
	SPS                  *SPS
	PPS                  *PPS
}

// Init this class.
func (me *AVC) Init(info *av.Information, logger log.ILogger) av.IMediaStreamTrackSource {
	me.EventDispatcher.Init(logger)
	me.logger = logger
	me.info = info
	me.infoframe = nil
	me.SPS = new(SPS).Init(info, logger)
	me.PPS = new(PPS).Init(me.SPS, logger)
	me.ctx.MimeType = "video/mp4"
	me.ctx.Codec = ""
	me.ctx.RefSampleDuration = uint32(float64(me.info.Timescale) * me.info.FrameRate.Den / me.info.FrameRate.Num)
	me.ctx.Flags.IsLeading = 0
	me.ctx.Flags.SampleDependsOn = 0
	me.ctx.Flags.SampleIsDependedOn = 0
	me.ctx.Flags.SampleHasRedundancy = 0
	me.ctx.Flags.IsNonSync = 0
	return me
}

// Kind returns the source name.
func (me *AVC) Kind() string {
	return "AVC"
}

// Context returns the source context.
func (me *AVC) Context() *av.Context {
	return &me.ctx
}

// SetInfoFrame stores the info frame for decoding.
func (me *AVC) SetInfoFrame(pkt *av.Packet) {
	me.infoframe = pkt
}

// GetInfoFrame returns the info frame.
func (me *AVC) GetInfoFrame() *av.Packet {
	return me.infoframe
}

// Sink a packet into the source.
func (me *AVC) Sink(pkt *av.Packet) {
	me.DispatchEvent(MediaEvent.New(MediaEvent.PACKET, me, pkt))
}

// Parse an AVC packet.
func (me *AVC) Parse(pkt *av.Packet) error {
	if pkt.Left() < 4 {
		err := fmt.Errorf("data not enough while parsing AVC packet")
		me.logger.Errorf("%v", err)
		return err
	}

	me.info.Timestamp = pkt.Timestamp

	pkt.Set("DataType", pkt.Payload[pkt.Position])
	pkt.Position++
	pkt.Set("CTS", uint32(pkt.Payload[pkt.Position])<<16|uint32(pkt.Payload[pkt.Position+1])<<8|uint32(pkt.Payload[pkt.Position+2]))
	pkt.Position += 3

	switch pkt.Get("DataType").(byte) {
	case SEQUENCE_HEADER:
		return me.parseDecoderConfigurationRecord(pkt)
	case NALU:
		return me.parseNalUnits(pkt)
	case END_OF_SEQUENCE:
		me.logger.Errorf("AVC sequence end")
	default:
		err := fmt.Errorf("unrecognized AVC packet type: 0x%02X", pkt.Get("DataType").(byte))
		me.logger.Errorf("%v", err)
		return err
	}

	return nil
}

func (me *AVC) parseDecoderConfigurationRecord(pkt *av.Packet) error {
	if pkt.Left() < 7 {
		err := fmt.Errorf("data not enough while parsing AVC decoder configuration record")
		me.logger.Errorf("%v", err)
		return err
	}

	me.infoframe = pkt
	me.AVCC = pkt.Payload[pkt.Position:]
	me.ConfigurationVersion = me.AVCC[0]
	me.ProfileIndication = me.AVCC[1]
	me.ProfileCompatibility = me.AVCC[2]
	me.LevelIndication = me.AVCC[3]

	if me.ConfigurationVersion != 1 {
		err := fmt.Errorf("invalid AVC configuration version: %d", me.ConfigurationVersion)
		me.logger.Errorf("%v", err)
		return err
	}

	me.NalLengthSize = uint32(me.AVCC[4]&0x03) + 1
	if me.NalLengthSize < 3 {
		err := fmt.Errorf("invalid NalLengthSize: %d", me.NalLengthSize)
		me.logger.Errorf("%v", err)
		return err
	}

	i := uint16(5)

	numOfSequenceParameterSets := int(me.AVCC[i] & 0x1F)
	i++

	for x := 0; x < numOfSequenceParameterSets; x++ {
		sequenceParameterSetLength := binary.BigEndian.Uint16(me.AVCC[i : i+2])
		i += 2
		if sequenceParameterSetLength == 0 {
			continue
		}

		err := me.SPS.parse(me.AVCC[i : i+sequenceParameterSetLength])
		if err != nil {
			// Ignore parsing issue, leave it to the decoder.
			me.logger.Warnf("%v", err)
			break
		}

		me.ctx.RefSampleDuration = uint32(float64(me.info.Timescale) * me.info.FrameRate.Den / me.info.FrameRate.Num)
		i += sequenceParameterSetLength
	}

	numOfPictureParameterSets := int(me.AVCC[i])
	i++

	for x := 0; x < numOfPictureParameterSets; x++ {
		pictureParameterSetLength := binary.BigEndian.Uint16(me.AVCC[i : i+2])
		i += 2
		if pictureParameterSetLength == 0 {
			continue
		}

		// PPS is useless for extracting video information.
		// err := me.PPS.parse(me.AVCC[i : i+pictureParameterSetLength])
		// if err != nil {
		// 	// Ignore parsing issue, leave it to the decoder.
		// 	me.logger.Warnf("%v", err)
		// 	break
		// }

		i += pictureParameterSetLength
	}

	me.ctx.Codec = me.SPS.Codec
	me.info.Codecs = append(me.info.Codecs, me.ctx.Codec)
	return nil
}

func (me *AVC) parseNalUnits(pkt *av.Packet) error {
	data := pkt.Payload[pkt.Position:]
	size := len(data)
	nalus := make([][]byte, 0)

	pkt.Set("DTS", me.info.Timestamp)
	pkt.Set("PTS", pkt.Get("CTS").(uint32)+pkt.Get("DTS").(uint32))
	pkt.Set("Data", data)

	for i := 0; i < size; /* void */ {
		if i+4 >= size {
			err := fmt.Errorf("data not enough while parsing AVC Nalus")
			me.logger.Errorf("%v", err)
			return err
		}

		naluSize := int(binary.BigEndian.Uint32(data[i : i+4]))
		if me.NalLengthSize == 3 { // NalLengthSize: 3 or 4 bytes
			naluSize >>= 8
		}
		if naluSize > size-int(me.NalLengthSize) {
			err := fmt.Errorf("malformed Nalus near timestamp %d", pkt.Get("DTS").(uint32))
			me.logger.Errorf("%v", err)
			return err
		}

		header := data[i+int(me.NalLengthSize)]
		pkt.Set("ForbiddenZeroBit", (header>>7)&0x01)
		pkt.Set("NalRefIdc", (header>>5)&0x03)
		pkt.Set("NalUnitType", header&0x1F)
		if pkt.Get("NalUnitType") == NAL_IDR_SLICE {
			pkt.Set("Keyframe", true)
		}
		if pkt.Get("ForbiddenZeroBit") != 0 {
			// me.logger.Warnf("Invalid NAL unit %d, skipping.", pkt.Get("NalUnitType"))
			// return fmt.Errorf("invalid")
		}

		nalu := data[i : i+int(me.NalLengthSize)+naluSize]
		nalus = append(nalus, nalu)

		i += int(me.NalLengthSize) + naluSize
	}

	pkt.Set("NALUs", nalus)
	return nil
}

func ebsp2rbsp(data []byte) []byte {
	i := 2
	n := len(data)

	dst := make([]byte, n)
	dst[0] = data[0]
	dst[1] = data[1]

	for j := 2; j < n; j++ {
		if data[j] == 0x03 && data[j-1] == 0x00 && data[j-2] == 0x00 {
			continue
		}

		dst[i] = data[j]
		i++
	}

	return dst[:i]
}
