package avc

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/studease/common/av"
	"github.com/studease/common/av/codec"
	"github.com/studease/common/av/utils"
	"github.com/studease/common/log"
)

// Static constants
const (
	MAX_PICTURE_COUNT      = 36
	MAX_SPS_COUNT          = 32
	MAX_PPS_COUNT          = 256
	MAX_LOG2_MAX_FRAME_NUM = 12 + 4
	MIN_LOG2_MAX_FRAME_NUM = 4
	EXTENDED_SAR           = 255
)

// Data types
const (
	SEQUENCE_HEADER = 0x00
	NALU            = 0x01
	END_OF_SEQUENCE = 0x02
)

// NAL types
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

var (
	pixelAspect = [17]av.Rational{
		{Num: 0, Den: 1},
		{Num: 1, Den: 1},
		{Num: 12, Den: 11},
		{Num: 10, Den: 11},
		{Num: 16, Den: 11},
		{Num: 40, Den: 33},
		{Num: 24, Den: 11},
		{Num: 20, Den: 11},
		{Num: 32, Den: 11},
		{Num: 80, Den: 33},
		{Num: 18, Den: 11},
		{Num: 15, Den: 11},
		{Num: 64, Den: 33},
		{Num: 160, Den: 99},
		{Num: 4, Den: 3},
		{Num: 3, Den: 2},
		{Num: 2, Den: 1},
	}
)

func init() {
	codec.Register(codec.AVC, Context{})
}

// Context implements IMediaContext
type Context struct {
	av.Context

	logger log.ILogger

	// Decoder Configuration Record
	AVCC                 []byte
	ConfigurationVersion byte
	ProfileIndication    byte
	ProfileCompatibility byte
	LevelIndication      byte
	NalLengthSize        uint32 // length_size_minus1 + 1
	SPS                  SPS
	PPS                  PPS

	// NAL Units
	ForbiddenZeroBit byte // 1 bits
	NalRefIdc        byte // 2 bits
	NalUnitType      byte // 5 bits
	NALUs            [][]byte
}

// Init this class
func (me *Context) Init(info *av.Information, logger log.ILogger) av.IMediaContext {
	me.Context.Init(info)
	me.logger = logger
	me.MimeType = "video/mp4"
	me.Flags.IsLeading = 0
	me.Flags.SampleHasRedundancy = 0
	me.Flags.IsNonSync = 0
	return me
}

// Codec returns the codec ID of this context
func (me *Context) Codec() av.Codec {
	return codec.AVC
}

// Parse an AVC packet
func (me *Context) Parse(p *av.Packet) error {
	if len(p.Payload) < 5 {
		err := fmt.Errorf("data not enough while parsing AVC packet")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	p.Context = me
	info := me.Information()
	info.Timestamp += p.Timestamp

	i := 0

	tmp := p.Payload[i]
	me.FrameType = tmp >> 4
	me.Context.Codec = tmp & 0x0F
	i++

	me.DataType = p.Payload[i]
	i++

	me.CTS = uint32(p.Payload[i])<<16 | uint32(p.Payload[i+1])<<8 | uint32(p.Payload[i+2])
	i += 3

	switch me.DataType {
	case SEQUENCE_HEADER:
		return me.parseDecoderConfigurationRecord(p.Timestamp, p.Payload[i:])

	case NALU:
		return me.parseNalUnits(p.Timestamp, p.Payload[i:])

	case END_OF_SEQUENCE:
		me.logger.Debugf(2, "AVC sequence end")

	default:
		err := fmt.Errorf("unrecognized AVC packet type: %02X", me.DataType)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	return nil
}

func (me *Context) parseDecoderConfigurationRecord(timestamp uint32, data []byte) error {
	if len(data) < 7 {
		err := fmt.Errorf("data not enough while parsing AVC decoder configuration record")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	me.AVCC = data
	me.ConfigurationVersion = data[0]
	me.ProfileIndication = data[1]
	me.ProfileCompatibility = data[2]
	me.LevelIndication = data[3]

	if me.ConfigurationVersion != 1 {
		err := fmt.Errorf("invalid AVC configuration version: %d", me.ConfigurationVersion)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	me.NalLengthSize = uint32(data[4]&0x03) + 1
	if me.NalLengthSize < 3 {
		err := fmt.Errorf("invalid NalLengthSize: %d", me.NalLengthSize)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	i := uint16(5)

	spsNum := int(data[i] & 0x1F)
	i++

	for x := 0; x < spsNum; x++ {
		n := binary.BigEndian.Uint16(data[i : i+2])
		if n == 0 {
			continue
		}

		i += 2

		err := me.parseSPS(timestamp, data[i:i+n])
		if err != nil {
			me.logger.Debugf(2, "%v", err)
			return err
		}

		i += n
	}

	ppsNum := int(data[i])
	i++

	for x := 0; x < ppsNum; x++ {
		n := binary.BigEndian.Uint16(data[i : i+2])
		if n == 0 {
			continue
		}

		i += 2

		err := me.parsePPS(timestamp, data[i:i+n])
		if err != nil {
			me.logger.Debugf(2, "%v", err)
			return err
		}

		i += n
	}

	return nil
}

func (me *Context) parseNalUnits(timestamp uint32, data []byte) error {
	size := len(data)
	info := me.Information()

	me.Keyframe = me.FrameType == av.KEYFRAME
	me.DTS = info.TimeBase + info.Timestamp
	me.PTS = me.CTS + me.DTS
	me.Data = data
	me.NALUs = make([][]byte, 0)

	for i := 0; i < size; /* void */ {
		if i+4 >= size {
			err := fmt.Errorf("data not enough while parsing AVC Nalus")
			me.logger.Debugf(2, "%v", err)
			return err
		}

		naluSize := int(binary.BigEndian.Uint32(data[i : i+4]))
		if me.NalLengthSize == 3 {
			naluSize >>= 8
		}

		i += 4

		if naluSize > size-int(me.NalLengthSize) {
			err := fmt.Errorf("malformed Nalus near timestamp %d", me.DTS)
			me.logger.Debugf(2, "%v", err)
			return err
		}

		nalu := data[i : i+naluSize]
		i += naluSize

		me.NalUnitType = nalu[0] & 0x1F
		if me.NalUnitType == NAL_IDR_SLICE {
			me.Keyframe = true
		}

		me.NALUs = append(me.NALUs, nalu)
	}

	return nil
}

func (me *Context) parseSPS(timestamp uint32, data []byte) error {
	if len(data) < 4 {
		err := fmt.Errorf("data not enough while parsing SPS")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	me.Codecs = "avc1." + hex.EncodeToString(data[1:4])

	info := me.Information()
	sps := &me.SPS
	gb := &sps.Golomb

	err := me.extractRBSP(gb, data)
	if err != nil {
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps.Data = gb.Buffer

	gb.ReadBits(8)

	sps.ProfileIdc = uint8(gb.ReadBits(8))
	sps.ConstraintSetFlags = byte(gb.ReadBits(1)) << 0  // constraint_set0_flag
	sps.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 1 // constraint_set1_flag
	sps.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 2 // constraint_set2_flag
	sps.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 3 // constraint_set3_flag
	sps.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 4 // constraint_set4_flag
	sps.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 5 // constraint_set5_flag
	sps.ReservedZero2Bits = byte(gb.ReadBits(2))
	sps.LevelIdc = byte(gb.ReadBits(8))
	sps.ID = uint32(gb.ReadUE())

	if sps.ID >= MAX_SPS_COUNT {
		err := fmt.Errorf("SPS ID %d out of range", sps.ID)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps.SeqScalingMatrixPresentFlag = 0
	sps.Vui.VideoFullRangeFlag = 1
	sps.Vui.MatrixCoefficients = av.COL_SPC_UNSPECIFIED

	if sps.ProfileIdc == 100 || // High profile
		sps.ProfileIdc == 110 || // High10 profile
		sps.ProfileIdc == 122 || // High422 profile
		sps.ProfileIdc == 244 || // High444 Predictive profile
		sps.ProfileIdc == 44 || // Cavlc444 profile
		sps.ProfileIdc == 83 || // Scalable Constrained High profile (SVC)
		sps.ProfileIdc == 86 || // Scalable High Intra profile (SVC)
		sps.ProfileIdc == 118 || // Stereo High profile (MVC)
		sps.ProfileIdc == 128 || // Multiview High profile (MVC)
		sps.ProfileIdc == 138 || // Multiview Depth High profile (MVCD)
		sps.ProfileIdc == 144 { // old High444 profile
		sps.ChromaFormatIdc = uint32(gb.ReadUE())
		if sps.ChromaFormatIdc > 3 {
			err := fmt.Errorf("bad sps.ChromaFormatIdc %d", sps.ChromaFormatIdc)
			me.logger.Debugf(2, "%v", err)
			return err
		} else if sps.ChromaFormatIdc == 3 {
			sps.SeparateColourPlaneFlag = uint8(gb.ReadBits(1))
			if sps.SeparateColourPlaneFlag != 0 {
				err := fmt.Errorf("separate color planes are not supported")
				me.logger.Debugf(2, "%v", err)
				return err
			}
		}

		sps.BitDepthLuma = uint32(gb.ReadUE()) + 8
		sps.BitDepthChroma = uint32(gb.ReadUE()) + 8
		if sps.BitDepthChroma != sps.BitDepthLuma {
			err := fmt.Errorf("different chroma and luma bit depth")
			me.logger.Debugf(2, "%v", err)
			return err
		}

		if sps.BitDepthLuma < 8 || sps.BitDepthLuma > 14 ||
			sps.BitDepthChroma < 8 || sps.BitDepthChroma > 14 {
			err := fmt.Errorf("illegal bit depth value (%d, %d)", sps.BitDepthLuma, sps.BitDepthChroma)
			me.logger.Debugf(2, "%v", err)
			return err
		}

		sps.TransformBypass = uint8(gb.ReadBits(1))

		sps.SeqScalingMatrixPresentFlag = uint8(gb.ReadBits(1))
		if sps.SeqScalingMatrixPresentFlag != 0 {
			n := 8
			if sps.ChromaFormatIdc == 3 {
				n += 4
			}

			for i := 0; i < n; i++ {
				if gb.ReadBits(1) != 0 { // seq_scaling_list_present_flag
					size := 16
					if i >= 6 {
						size = 64
					}

					last := 8
					next := 8

					for j := 0; j < size; j++ {
						delta := int(gb.ReadSE())
						next = (last + delta + 256) % 256
						if next != 0 {
							last = next
						}
					}
				}
			}
		}
	} else {
		sps.ChromaFormatIdc = 1
		sps.BitDepthLuma = 8
		sps.BitDepthChroma = 8
	}

	sps.Log2MaxFrameNum = uint32(gb.ReadUE()) + 4
	if sps.Log2MaxFrameNum < MIN_LOG2_MAX_FRAME_NUM || sps.Log2MaxFrameNum > MAX_LOG2_MAX_FRAME_NUM {
		err := fmt.Errorf("log2_max_frame_num_minus4 out of range (0-12): %d", sps.Log2MaxFrameNum-4)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps.PocType = uint32(gb.ReadUE())
	if sps.PocType == 0 {
		sps.Log2MaxPocLsb = uint32(gb.ReadUE()) + 4
		if sps.Log2MaxPocLsb > 16 {
			err := fmt.Errorf("log2_max_poc_lsb (%d) is out of range", sps.Log2MaxPocLsb)
			me.logger.Debugf(2, "%v", err)
			return err
		}
	} else if sps.PocType == 1 {
		sps.DeltaPicOrderAlwaysZeroFlag = byte(gb.ReadBits(1))
		sps.OffsetForNonRefPic = uint32(gb.ReadUE())
		sps.OffsetForTopToBottomField = uint32(gb.ReadUE())
		sps.NumRefFramesInPocCycle = uint32(gb.ReadUE())

		n := uint32(len(sps.OffsetForRefFrame))
		if sps.NumRefFramesInPocCycle >= n {
			err := fmt.Errorf("poc_cycle_length overflow %d", sps.NumRefFramesInPocCycle)
			me.logger.Debugf(2, "%v", err)
			return err
		}

		for i := uint32(0); i < sps.NumRefFramesInPocCycle; i++ {
			sps.OffsetForRefFrame[i] = uint16(gb.ReadUE())
		}
	} else if sps.PocType != 2 {
		err := fmt.Errorf("illegal POC type %d", sps.PocType)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps.MaxNumRefFrames = uint32(gb.ReadUE())
	if sps.MaxNumRefFrames > MAX_PICTURE_COUNT-2 || sps.MaxNumRefFrames > 16 {
		err := fmt.Errorf("too many reference frames %d", sps.MaxNumRefFrames)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps.GapsInFrameNumValueAllowedFlag = uint8(gb.ReadBits(1))
	sps.PicWidth = uint32(gb.ReadUE()) + 1
	sps.PicHeight = uint32(gb.ReadUE()) + 1
	if sps.PicWidth >= uint32(utils.MAX_INT32/16) || sps.PicHeight >= uint32(utils.MAX_INT32/16) ||
		me.checkImageSize(16*sps.PicWidth, 16*sps.PicHeight) == false {
		err := fmt.Errorf("pic_width or pic_height overflow")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps.FrameMbsOnlyFlag = uint8(gb.ReadBits(1))
	if sps.FrameMbsOnlyFlag == 0 {
		sps.MbAdaptiveFrameFieldFlag = byte(gb.ReadBits(1))
	} else {
		sps.MbAdaptiveFrameFieldFlag = 0
	}

	info.CodecWidth = 16 * sps.PicWidth
	info.CodecHeight = 16 * sps.PicHeight * uint32(2-sps.FrameMbsOnlyFlag)

	sps.Direct8x8InferenceFlag = uint8(gb.ReadBits(1))
	sps.FrameCroppingFlag = uint8(gb.ReadBits(1))
	if sps.FrameCroppingFlag != 0 {
		var (
			vsub uint32
			hsub uint32
		)

		cropLeft := uint32(gb.ReadUE())
		cropRight := uint32(gb.ReadUE())
		cropTop := uint32(gb.ReadUE())
		cropBottom := uint32(gb.ReadUE())

		if sps.ChromaFormatIdc == 1 {
			vsub = 1
		}
		if sps.ChromaFormatIdc == 1 || sps.ChromaFormatIdc == 2 {
			hsub = 1
		}
		stepX := uint32(1 << hsub)
		stepY := uint32(2-sps.FrameMbsOnlyFlag) << vsub

		if cropLeft > uint32(utils.MAX_INT32/4)/stepX ||
			cropRight > uint32(utils.MAX_INT32/4)/stepX ||
			cropTop > uint32(utils.MAX_INT32/4)/stepY ||
			cropBottom > uint32(utils.MAX_INT32/4)/stepY ||
			(cropLeft+cropRight)*stepX >= info.CodecWidth ||
			(cropTop+cropBottom)*stepY >= info.CodecHeight {
			err := fmt.Errorf("invalid crop values %d %d %d %d / %d %d",
				cropLeft, cropRight, cropTop, cropBottom, info.CodecWidth, info.CodecHeight)
			me.logger.Debugf(2, "%v", err)
			return err
		}

		sps.FrameCropLeftOffset = uint32(cropLeft * stepX)
		sps.FrameCropRightOffset = uint32(cropRight * stepX)
		sps.FrameCropTopOffset = uint32(cropTop * stepY)
		sps.FrameCropBottomOffset = uint32(cropBottom * stepY)
	} else {
		sps.FrameCropLeftOffset = 0
		sps.FrameCropRightOffset = 0
		sps.FrameCropTopOffset = 0
		sps.FrameCropBottomOffset = 0
	}

	sps.VuiParametersPresentFlag = uint8(gb.ReadBits(1))
	if sps.VuiParametersPresentFlag != 0 {
		err := me.decodeVuiParameters(gb)
		if err != nil {
			return err
		}

		if sps.Vui.TimingInfoPresentFlag != 0 {
			info.FrameRate.Num = float64(sps.Vui.TimeScale)
			info.FrameRate.Den = float64(sps.Vui.NumUnitsInTick * 2)
		}

		info.RefSampleDuration = uint32(float64(info.Timescale) * info.FrameRate.Den / info.FrameRate.Num)
	}

	info.CodecWidth -= sps.FrameCropLeftOffset + sps.FrameCropRightOffset
	info.CodecHeight -= sps.FrameCropTopOffset + sps.FrameCropBottomOffset

	info.Width = info.CodecWidth
	info.Height = info.CodecHeight
	if sps.Vui.Sar.Den != 0 {
		info.Width *= uint32(sps.Vui.Sar.Num / sps.Vui.Sar.Den)
	}

	return nil
}

func (me *Context) checkImageSize(w uint32, h uint32) bool {
	return w != 0 && h != 0 && (w+128)*(h+128) < uint32(utils.MAX_INT32/8)
}

func (me *Context) decodeVuiParameters(gb *utils.Golomb) error {
	vui := &me.SPS.Vui

	vui.AspectRatioInfoPresentFlag = uint8(gb.ReadBits(1))
	if vui.AspectRatioInfoPresentFlag != 0 {
		vui.AspectRatioIdc = uint8(gb.ReadBits(8))
		if vui.AspectRatioIdc == EXTENDED_SAR {
			vui.Sar.Num = float64(gb.ReadBits(16))
			vui.Sar.Den = float64(gb.ReadBits(16))
		} else if vui.AspectRatioIdc < uint8(len(pixelAspect)) {
			vui.Sar = pixelAspect[vui.AspectRatioIdc]
		} else {
			err := fmt.Errorf("illegal aspect ratio")
			me.logger.Debugf(2, "%v", err)
			return err
		}
	} else {
		vui.Sar.Num = 0
		vui.Sar.Den = 0
	}

	vui.OverscanInfoPresentFlag = uint8(gb.ReadBits(1))
	if vui.OverscanInfoPresentFlag != 0 {
		vui.OverscanAppropriateFlag = uint8(gb.ReadBits(1))
	}

	vui.VideoSignalTypePresentFlag = uint8(gb.ReadBits(1))
	if vui.VideoSignalTypePresentFlag != 0 {
		vui.VideoFormat = uint8(gb.ReadBits(3))
		vui.VideoFullRangeFlag = uint8(gb.ReadBits(1))
		vui.ColourDescriptionPresentFlag = uint8(gb.ReadBits(1))

		if vui.ColourDescriptionPresentFlag != 0 {
			vui.ColourPrimaries = uint8(gb.ReadBits(8))
			vui.TransferCharacteristics = uint8(gb.ReadBits(8))
			vui.MatrixCoefficients = uint8(gb.ReadBits(8))

			if vui.ColourPrimaries >= av.COL_PRI_NB {
				vui.ColourPrimaries = av.COL_PRI_UNSPECIFIED
			}
			if vui.TransferCharacteristics >= av.COL_TRC_NB {
				vui.TransferCharacteristics = av.COL_TRC_UNSPECIFIED
			}
			if vui.MatrixCoefficients >= av.COL_SPC_NB {
				vui.MatrixCoefficients = av.COL_SPC_UNSPECIFIED
			}
		}
	}

	vui.ChromaLocInfoPresentFlag = uint8(gb.ReadBits(1))
	if vui.ChromaLocInfoPresentFlag != 0 {
		vui.ChromaSampleLocTypeTopField = uint32(gb.ReadUE())
		vui.ChromaSampleLocTypeBottomField = uint32(gb.ReadUE())
	}

	if gb.ReadBits(1) != 0 && gb.Left() < 10 {
		err := fmt.Errorf("truncated VUI")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	vui.TimingInfoPresentFlag = uint8(gb.ReadBits(1))
	if vui.TimingInfoPresentFlag != 0 {
		vui.NumUnitsInTick = uint32(gb.ReadBitsLong(32))
		vui.TimeScale = uint32(gb.ReadBitsLong(32))

		if vui.NumUnitsInTick == 0 || vui.TimeScale == 0 {
			me.logger.Debugf(2, "time_scale/num_units_in_tick invalid or unsupported (%u/%u)", vui.TimeScale, vui.NumUnitsInTick)
			vui.TimingInfoPresentFlag = 0
		}

		vui.FixedFrameRateFlag = uint8(gb.ReadBits(1))
	}

	vui.NalHrdParametersPresentFlag = uint8(gb.ReadBits(1))
	if vui.NalHrdParametersPresentFlag != 0 {
		err := me.decodeHrdParameters(gb, &vui.NalHrd)
		if err != nil {
			return err
		}
	}

	vui.VclHrdParametersPresentFlag = uint8(gb.ReadBits(1))
	if vui.VclHrdParametersPresentFlag != 0 {
		err := me.decodeHrdParameters(gb, &vui.VclHrd)
		if err != nil {
			return err
		}
	}

	if vui.NalHrdParametersPresentFlag != 0 || vui.VclHrdParametersPresentFlag != 0 {
		vui.LowDelayHrdFlag = uint8(gb.ReadBits(1))
	}

	vui.PicStructPresentFlag = uint8(gb.ReadBits(1))
	if gb.Left() == 0 {
		return nil
	}

	vui.BitstreamRestrictionFlag = uint8(gb.ReadBits(1))
	if vui.BitstreamRestrictionFlag != 0 {
		vui.MotionVectorsOverPicBoundariesFlag = uint8(gb.ReadBits(1))
		vui.MaxBytesPerPicDenom = uint32(gb.ReadUE())
		vui.MaxBitsPerMbDenom = uint32(gb.ReadUE())
		vui.Log2MaxMvLengthHorizontal = uint32(gb.ReadUE())
		vui.Log2MaxMvLengthVertical = uint32(gb.ReadUE())
		vui.MaxNumReorderFrames = uint32(gb.ReadUE())
		vui.MaxDecFrameBuffering = uint32(gb.ReadUE())

		if gb.Left() < 0 {
			vui.MaxNumReorderFrames = 0
			vui.BitstreamRestrictionFlag = 0
		}

		if vui.MaxNumReorderFrames > 16 /* max_dec_frame_buffering || max_dec_frame_buffering > 16 */ {
			vui.MaxNumReorderFrames = 16
			err := fmt.Errorf("clipping illegal MaxNumReorderFrames %d", vui.MaxNumReorderFrames)
			me.logger.Debugf(2, "%v", err)
			return err
		}
	}

	return nil
}

func (me *Context) decodeHrdParameters(gb *utils.Golomb, hrd *HRD) error {
	hrd.CpbCnt = uint32(gb.ReadUE()) + 1
	if hrd.CpbCnt > 32 {
		err := fmt.Errorf("invalid hrd.CpbCnt %d", hrd.CpbCnt)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	hrd.BitRateScale = uint8(gb.ReadBits(4))
	hrd.CpbSizeScale = uint8(gb.ReadBits(4))

	for i := uint32(0); i < hrd.CpbCnt; i++ {
		hrd.BitRateValue[i] = uint32(gb.ReadUE())
		hrd.CpbSizeValue[i] = uint32(gb.ReadUE())
		hrd.CbrFlag |= uint32(gb.ReadBits(1)) << i
	}

	hrd.InitialCpbRemovalDelayLength = uint32(gb.ReadBits(5)) + 1
	hrd.CpbRemovalDelayLength = uint32(gb.ReadBits(5)) + 1
	hrd.DpbOutputDelayLength = uint32(gb.ReadBits(5)) + 1
	hrd.TimeOffsetLength = uint32(gb.ReadBits(5))

	return nil
}

func (me *Context) extractRBSP(gb *utils.Golomb, data []byte) error {
	i := 2
	n := len(data)

	tmp := make([]byte, n)
	tmp[0] = data[0]
	tmp[1] = data[1]

	for j := 2; j < n; j++ {
		if data[j] == 0x03 && data[j-1] == 0x00 && data[j-2] == 0x00 {
			continue
		}

		tmp[i] = data[j]
		i++
	}

	if gb.Init(tmp[:i]) == nil {
		return fmt.Errorf("failed to init Golomb")
	}

	return nil
}

func (me *Context) parsePPS(timestamp uint32, data []byte) error {
	err := me.extractRBSP(&me.PPS.Golomb, data)
	if err != nil {
		me.logger.Debugf(2, "%v", err)
		return err
	}

	sps := &me.SPS
	pps := &me.PPS
	gb := &pps.Golomb
	pps.Data = gb.Buffer

	pps.ID = uint32(gb.ReadUE())
	if pps.ID >= MAX_PPS_COUNT {
		err := fmt.Errorf("PPS ID %d out of range", pps.ID)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	pps.SpsID = uint32(gb.ReadUE())
	if pps.SpsID >= MAX_SPS_COUNT {
		err := fmt.Errorf("SPS ID %d out of range", pps.SpsID)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	if sps.BitDepthLuma > 14 {
		err := fmt.Errorf("invalid BitDepthLuma %d", sps.BitDepthLuma)
		me.logger.Debugf(2, "%v", err)
		return err
	} else if sps.BitDepthLuma == 11 || sps.BitDepthLuma == 13 {
		err := fmt.Errorf("unimplemented BitDepthLuma %d", sps.BitDepthLuma)
		me.logger.Debugf(2, "%v", err)
		return err
	}

	pps.EntropyCodingModeFlag = uint8(gb.ReadBits(1))
	pps.PicOrderPresentFlag = uint8(gb.ReadBits(1))
	pps.NumSliceGroups = uint32(gb.ReadUE()) + 1
	if pps.NumSliceGroups > 1 {
		pps.SliceGroupMapType = uint32(gb.ReadUE())
		me.logger.Debugf(2, "FMO not supported")

		switch pps.SliceGroupMapType {
		case 0:
			/*
				for (i = 0; i <= num_slice_groups_minus1; i++)  |   |      |
				run_length[i]                                   |1  |ue(v) |
			*/

		case 2:
			/*
				for (i = 0; i < num_slice_groups_minus1; i++) { |   |      |
					top_left_mb[i]                              |1  |ue(v) |
					bottom_right_mb[i]                          |1  |ue(v) |
				}                                               |   |      |
			*/

		case 3:
			fallthrough
		case 4:
			fallthrough
		case 5:
			/*
				slice_group_change_direction_flag               |1  |u(1)  |
				slice_group_change_rate_minus1                  |1  |ue(v) |
			*/

		case 6:
			/*
				slice_group_id_cnt_minus1                       |1  |ue(v) |
				for (i = 0; i <= slice_group_id_cnt_minus1; i++)|   |      |
					slice_group_id[i]                           |1  |u(v)  |
			*/

		}
	}

	pps.NumRefIdx[0] = uint32(gb.ReadUE()) + 1
	pps.NumRefIdx[1] = uint32(gb.ReadUE()) + 1
	if pps.NumRefIdx[0]-1 > 32-1 || pps.NumRefIdx[1]-1 > 32-1 {
		err := fmt.Errorf("reference overflow (pps)")
		me.logger.Debugf(2, "%v", err)
		return err
	}

	qpBdOffset := 6 * (sps.BitDepthLuma - 8)

	pps.WeightedPredFlag = uint8(gb.ReadBits(1))
	pps.WeightedBipredIdc = uint8(gb.ReadBits(2))
	pps.PicInitQp = uint32(gb.ReadSE()) + 26 + qpBdOffset
	pps.PicInitQs = uint32(gb.ReadSE()) + 26 + qpBdOffset
	pps.ChromaQpIndexOffset[0] = int32(gb.ReadSE())
	pps.DeblockingFilterControlPresentFlag = uint8(gb.ReadBits(1))
	pps.ConstrainedIntraPredFlag = uint8(gb.ReadBits(1))
	pps.RedundantPicCntPresentFlag = uint8(gb.ReadBits(1))

	pps.Transform8x8ModeFlag = 0

	if gb.Left() > 0 && me.moreRBSPInPPS(sps) {
		pps.Transform8x8ModeFlag = uint8(gb.ReadBits(1))
		pps.PicScalingMatrixPresentFlag = uint8(gb.ReadBits(1))
		if pps.PicScalingMatrixPresentFlag != 0 {
			n := 2
			if sps.ChromaFormatIdc == 3 {
				n = 6
			}

			gb.ReadBits(6 + n*int(pps.Transform8x8ModeFlag))
		}

		pps.ChromaQpIndexOffset[1] = int32(gb.ReadSE()) // second_chroma_qp_index_offset
	} else {
		pps.ChromaQpIndexOffset[1] = pps.ChromaQpIndexOffset[0]
	}

	if pps.ChromaQpIndexOffset[0] != pps.ChromaQpIndexOffset[1] {
		pps.ChromaQpDiff = 1
	}

	return nil
}

func (me *Context) moreRBSPInPPS(sps *SPS) bool {
	if (sps.ProfileIdc == 66 || sps.ProfileIdc == 77 || sps.ProfileIdc == 88) && (sps.ConstraintSetFlags&7) != 0 {
		err := fmt.Errorf("current profile doesn't provide more RBSP data in PPS, skipping")
		me.logger.Debugf(2, "%v", err)
		return false
	}

	return true
}
