package avc

import (
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

// SPS (Sequence Parameter Set)
type SPS struct {
	logger log.ILogger
	info   *av.Information
	Codec  string

	ProfileIdc                     uint8
	ConstraintSetFlags             uint8 // 6 bits
	ReservedZero2Bits              uint8 // 2 bits, equal to 0
	LevelIdc                       uint8
	ID                             uint32 // seq_parameter_set_id
	ChromaFormatIdc                uint32
	SeparateColourPlaneFlag        uint8  // 1 bits
	BitDepthLuma                   uint32 // bit_depth_luma_minus8 + 8
	BitDepthChroma                 uint32 // bit_depth_chroma_minus8 + 8
	TransformBypass                uint8  // 1 bits, qpprime_y_zero_transform_bypass_flag
	SeqScalingMatrixPresentFlag    uint8  // 1 bits
	Log2MaxFrameNum                uint32 // log2_max_frame_num_minus4 + 4
	PocType                        uint32 // pic_order_cnt_type
	Log2MaxPocLsb                  uint32 // log2_max_pic_order_cnt_lsb_minus4 + 4
	DeltaPicOrderAlwaysZeroFlag    uint8  // 1 bits
	OffsetForNonRefPic             uint32
	OffsetForTopToBottomField      uint32
	NumRefFramesInPocCycle         uint32 // num_ref_frames_in_pic_order_cnt_cycle
	OffsetForRefFrame              [256]uint16
	MaxNumRefFrames                uint32
	GapsInFrameNumValueAllowedFlag uint8  // 1 bits
	PicWidth                       uint32 // pic_width_in_mbs_minus1 + 1
	PicHeight                      uint32 // pic_height_in_map_units_minus1 + 1
	FrameMbsOnlyFlag               uint8  // 1 bits
	MbAdaptiveFrameFieldFlag       uint8  // 1 bits
	Direct8x8InferenceFlag         uint8  // 1 bits
	FrameCroppingFlag              uint8  // 1 bits
	FrameCropLeftOffset            uint32
	FrameCropRightOffset           uint32
	FrameCropTopOffset             uint32
	FrameCropBottomOffset          uint32
	VuiParametersPresentFlag       uint8 // 1 bits
	Vui                            VUI
	Data                           []byte
}

// Init this class.
func (me *SPS) Init(info *av.Information, logger log.ILogger) *SPS {
	me.info = info
	me.logger = logger
	me.Vui.Init(logger)
	return me
}

func (me *SPS) parse(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("data not enough while parsing SPS")
	}

	me.Codec = "avc1." + hex.EncodeToString(data[1:4])

	rbsp := ebsp2rbsp(data)
	gb := new(utils.Golomb).Init(rbsp)
	me.Data = rbsp

	gb.ReadBits(8)

	me.ProfileIdc = uint8(gb.ReadBits(8))
	me.ConstraintSetFlags = byte(gb.ReadBits(1)) << 0  // constraint_set0_flag
	me.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 1 // constraint_set1_flag
	me.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 2 // constraint_set2_flag
	me.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 3 // constraint_set3_flag
	me.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 4 // constraint_set4_flag
	me.ConstraintSetFlags |= byte(gb.ReadBits(1)) << 5 // constraint_set5_flag
	me.ReservedZero2Bits = byte(gb.ReadBits(2))
	me.LevelIdc = byte(gb.ReadBits(8))
	me.ID = uint32(gb.ReadUE())

	if me.ID >= MAX_SPS_COUNT {
		return fmt.Errorf("SPS ID(0x%02X) out of range", me.ID)
	}

	me.SeqScalingMatrixPresentFlag = 0
	me.Vui.VideoFullRangeFlag = 1
	me.Vui.MatrixCoefficients = codec.COL_SPC_UNSPECIFIED

	if me.ProfileIdc == 100 || // High profile
		me.ProfileIdc == 110 || // High10 profile
		me.ProfileIdc == 122 || // High422 profile
		me.ProfileIdc == 244 || // High444 Predictive profile
		me.ProfileIdc == 44 || // Cavlc444 profile
		me.ProfileIdc == 83 || // Scalable Constrained High profile (SVC)
		me.ProfileIdc == 86 || // Scalable High Intra profile (SVC)
		me.ProfileIdc == 118 || // Stereo High profile (MVC)
		me.ProfileIdc == 128 || // Multiview High profile (MVC)
		me.ProfileIdc == 138 || // Multiview Depth High profile (MVCD)
		me.ProfileIdc == 144 { // old High444 profile
		me.ChromaFormatIdc = uint32(gb.ReadUE())
		if me.ChromaFormatIdc > 3 {
			return fmt.Errorf("bad ChromaFormatIdc %d", me.ChromaFormatIdc)
		} else if me.ChromaFormatIdc == 3 {
			me.SeparateColourPlaneFlag = uint8(gb.ReadBits(1))
			if me.SeparateColourPlaneFlag != 0 {
				return fmt.Errorf("separate color planes are not supported")
			}
		}

		me.BitDepthLuma = uint32(gb.ReadUE()) + 8
		me.BitDepthChroma = uint32(gb.ReadUE()) + 8
		if me.BitDepthChroma != me.BitDepthLuma {
			return fmt.Errorf("different chroma and luma bit depth")
		}

		if me.BitDepthLuma < 8 || me.BitDepthLuma > 14 ||
			me.BitDepthChroma < 8 || me.BitDepthChroma > 14 {
			return fmt.Errorf("illegal bit depth value (%d, %d)", me.BitDepthLuma, me.BitDepthChroma)
		}

		me.TransformBypass = uint8(gb.ReadBits(1))

		me.SeqScalingMatrixPresentFlag = uint8(gb.ReadBits(1))
		if me.SeqScalingMatrixPresentFlag != 0 {
			n := 8
			if me.ChromaFormatIdc == 3 {
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
		me.ChromaFormatIdc = 1
		me.BitDepthLuma = 8
		me.BitDepthChroma = 8
	}

	me.Log2MaxFrameNum = uint32(gb.ReadUE()) + 4
	if me.Log2MaxFrameNum < MIN_LOG2_MAX_FRAME_NUM || me.Log2MaxFrameNum > MAX_LOG2_MAX_FRAME_NUM {
		return fmt.Errorf("log2_max_frame_num_minus4 out of range (0-12): %d", me.Log2MaxFrameNum-4)
	}

	me.PocType = uint32(gb.ReadUE())
	if me.PocType == 0 {
		me.Log2MaxPocLsb = uint32(gb.ReadUE()) + 4
		if me.Log2MaxPocLsb > 16 {
			return fmt.Errorf("log2_max_poc_lsb (%d) is out of range", me.Log2MaxPocLsb)
		}
	} else if me.PocType == 1 {
		me.DeltaPicOrderAlwaysZeroFlag = byte(gb.ReadBits(1))
		me.OffsetForNonRefPic = uint32(gb.ReadUE())
		me.OffsetForTopToBottomField = uint32(gb.ReadUE())
		me.NumRefFramesInPocCycle = uint32(gb.ReadUE())

		n := uint32(len(me.OffsetForRefFrame))
		if me.NumRefFramesInPocCycle >= n {
			return fmt.Errorf("poc_cycle_length overflow %d", me.NumRefFramesInPocCycle)
		}

		for i := uint32(0); i < me.NumRefFramesInPocCycle; i++ {
			me.OffsetForRefFrame[i] = uint16(gb.ReadUE())
		}
	} else if me.PocType != 2 {
		return fmt.Errorf("illegal POC type %d", me.PocType)
	}

	me.MaxNumRefFrames = uint32(gb.ReadUE())
	if me.MaxNumRefFrames > MAX_PICTURE_COUNT-2 || me.MaxNumRefFrames > 16 {
		return fmt.Errorf("too many reference frames %d", me.MaxNumRefFrames)
	}

	me.GapsInFrameNumValueAllowedFlag = uint8(gb.ReadBits(1))
	me.PicWidth = uint32(gb.ReadUE()) + 1
	me.PicHeight = uint32(gb.ReadUE()) + 1
	if me.PicWidth >= uint32(utils.MAX_INT32/16) || me.PicHeight >= uint32(utils.MAX_INT32/16) ||
		me.checkImageSize(16*me.PicWidth, 16*me.PicHeight) == false {
		return fmt.Errorf("pic_width or pic_height overflow")
	}

	me.FrameMbsOnlyFlag = uint8(gb.ReadBits(1))
	if me.FrameMbsOnlyFlag == 0 {
		me.MbAdaptiveFrameFieldFlag = byte(gb.ReadBits(1))
	} else {
		me.MbAdaptiveFrameFieldFlag = 0
	}

	me.info.CodecWidth = 16 * me.PicWidth
	me.info.CodecHeight = 16 * me.PicHeight * uint32(2-me.FrameMbsOnlyFlag)

	me.Direct8x8InferenceFlag = uint8(gb.ReadBits(1))
	me.FrameCroppingFlag = uint8(gb.ReadBits(1))
	if me.FrameCroppingFlag != 0 {
		var (
			vsub uint32
			hsub uint32
		)

		cropLeft := uint32(gb.ReadUE())
		cropRight := uint32(gb.ReadUE())
		cropTop := uint32(gb.ReadUE())
		cropBottom := uint32(gb.ReadUE())

		if me.ChromaFormatIdc == 1 {
			vsub = 1
		}
		if me.ChromaFormatIdc == 1 || me.ChromaFormatIdc == 2 {
			hsub = 1
		}
		stepX := uint32(1 << hsub)
		stepY := uint32(2-me.FrameMbsOnlyFlag) << vsub

		if cropLeft > uint32(utils.MAX_INT32/4)/stepX ||
			cropRight > uint32(utils.MAX_INT32/4)/stepX ||
			cropTop > uint32(utils.MAX_INT32/4)/stepY ||
			cropBottom > uint32(utils.MAX_INT32/4)/stepY ||
			(cropLeft+cropRight)*stepX >= me.info.CodecWidth ||
			(cropTop+cropBottom)*stepY >= me.info.CodecHeight {
			return fmt.Errorf("invalid crop values, l=%d, r=%d, t=%d, b=%d, w=%d, h=%d",
				cropLeft, cropRight, cropTop, cropBottom, me.info.CodecWidth, me.info.CodecHeight)
		}

		me.FrameCropLeftOffset = uint32(cropLeft * stepX)
		me.FrameCropRightOffset = uint32(cropRight * stepX)
		me.FrameCropTopOffset = uint32(cropTop * stepY)
		me.FrameCropBottomOffset = uint32(cropBottom * stepY)
	} else {
		me.FrameCropLeftOffset = 0
		me.FrameCropRightOffset = 0
		me.FrameCropTopOffset = 0
		me.FrameCropBottomOffset = 0
	}

	me.VuiParametersPresentFlag = uint8(gb.ReadBits(1))
	if me.VuiParametersPresentFlag != 0 {
		err := me.Vui.parse(gb)
		if err != nil {
			// Ignore parsing issue, leave it to the decoder.
			// Fall through here, to calc the other info.
			me.logger.Warnf("%v", err)
		}

		if me.Vui.TimingInfoPresentFlag != 0 {
			me.info.FrameRate.Num = float64(me.Vui.TimeScale)
			me.info.FrameRate.Den = float64(me.Vui.NumUnitsInTick * 2)
		}
	}

	me.info.CodecWidth -= me.FrameCropLeftOffset + me.FrameCropRightOffset
	me.info.CodecHeight -= me.FrameCropTopOffset + me.FrameCropBottomOffset

	me.info.Width = me.info.CodecWidth
	me.info.Height = me.info.CodecHeight
	if me.Vui.Sar.Den > 1 {
		me.info.Width *= uint32(me.Vui.Sar.Num / me.Vui.Sar.Den)
	}

	return nil
}

func (me *SPS) checkImageSize(w uint32, h uint32) bool {
	return w != 0 && h != 0 && (w+128)*(h+128) < uint32(utils.MAX_INT32/8)
}

// VUI (Video Usability Information)
type VUI struct {
	logger log.ILogger

	AspectRatioInfoPresentFlag         uint8 // 1 bits
	AspectRatioIdc                     uint8
	Sar                                av.Rational
	OverscanInfoPresentFlag            uint8 // 1 bits
	OverscanAppropriateFlag            uint8 // 1 bits
	VideoSignalTypePresentFlag         uint8 // 1 bits
	VideoFormat                        uint8 // 3 bits
	VideoFullRangeFlag                 uint8 // 1 bits
	ColourDescriptionPresentFlag       uint8 // 1 bits
	ColourPrimaries                    uint8
	TransferCharacteristics            uint8
	MatrixCoefficients                 uint8
	ChromaLocInfoPresentFlag           uint8 // 1 bits
	ChromaSampleLocTypeTopField        uint32
	ChromaSampleLocTypeBottomField     uint32
	TimingInfoPresentFlag              byte // 1 bits
	NumUnitsInTick                     uint32
	TimeScale                          uint32
	FixedFrameRateFlag                 uint8 // 1 bits
	NalHrdParametersPresentFlag        uint8 // 1 bits
	NalHrd                             HRD
	VclHrdParametersPresentFlag        uint8 // 1 bits
	VclHrd                             HRD
	LowDelayHrdFlag                    uint8 // 1 bits
	PicStructPresentFlag               uint8 // 1 bits
	BitstreamRestrictionFlag           uint8 // 1 bits
	MotionVectorsOverPicBoundariesFlag uint8 // 1 bits
	MaxBytesPerPicDenom                uint32
	MaxBitsPerMbDenom                  uint32
	Log2MaxMvLengthHorizontal          uint32
	Log2MaxMvLengthVertical            uint32
	MaxNumReorderFrames                uint32
	MaxDecFrameBuffering               uint32
}

// Init this class.
func (me *VUI) Init(logger log.ILogger) *VUI {
	me.logger = logger
	me.Sar.Init(0, 1)
	me.NalHrd.Init(logger)
	me.VclHrd.Init(logger)
	return me
}

func (me *VUI) parse(gb *utils.Golomb) error {
	me.AspectRatioInfoPresentFlag = uint8(gb.ReadBits(1))
	if me.AspectRatioInfoPresentFlag != 0 {
		me.AspectRatioIdc = uint8(gb.ReadBits(8))
		if me.AspectRatioIdc == EXTENDED_SAR {
			me.Sar.Num = float64(gb.ReadBits(16))
			me.Sar.Den = float64(gb.ReadBits(16))
		} else if me.AspectRatioIdc < uint8(len(pixelAspect)) {
			me.Sar = pixelAspect[me.AspectRatioIdc]
		} else {
			return fmt.Errorf("illegal aspect ratio")
		}
	} else {
		me.Sar.Num = 0
		me.Sar.Den = 0
	}

	me.OverscanInfoPresentFlag = uint8(gb.ReadBits(1))
	if me.OverscanInfoPresentFlag != 0 {
		me.OverscanAppropriateFlag = uint8(gb.ReadBits(1))
	}

	me.VideoSignalTypePresentFlag = uint8(gb.ReadBits(1))
	if me.VideoSignalTypePresentFlag != 0 {
		me.VideoFormat = uint8(gb.ReadBits(3))
		me.VideoFullRangeFlag = uint8(gb.ReadBits(1))
		me.ColourDescriptionPresentFlag = uint8(gb.ReadBits(1))

		if me.ColourDescriptionPresentFlag != 0 {
			me.ColourPrimaries = uint8(gb.ReadBits(8))
			me.TransferCharacteristics = uint8(gb.ReadBits(8))
			me.MatrixCoefficients = uint8(gb.ReadBits(8))

			if me.ColourPrimaries >= codec.COL_PRI_NB {
				me.ColourPrimaries = codec.COL_PRI_UNSPECIFIED
			}
			if me.TransferCharacteristics >= codec.COL_TRC_NB {
				me.TransferCharacteristics = codec.COL_TRC_UNSPECIFIED
			}
			if me.MatrixCoefficients >= codec.COL_SPC_NB {
				me.MatrixCoefficients = codec.COL_SPC_UNSPECIFIED
			}
		}
	}

	me.ChromaLocInfoPresentFlag = uint8(gb.ReadBits(1))
	if me.ChromaLocInfoPresentFlag != 0 {
		me.ChromaSampleLocTypeTopField = uint32(gb.ReadUE())
		me.ChromaSampleLocTypeBottomField = uint32(gb.ReadUE())
	}

	if gb.ReadBits(1) != 0 && gb.Left() < 10 {
		return fmt.Errorf("truncated VUI")
	}

	me.TimingInfoPresentFlag = uint8(gb.ReadBits(1))
	if me.TimingInfoPresentFlag != 0 {
		me.NumUnitsInTick = uint32(gb.ReadBitsLong(32))
		me.TimeScale = uint32(gb.ReadBitsLong(32))

		if me.NumUnitsInTick == 0 || me.TimeScale == 0 {
			me.logger.Warnf("time_scale/num_units_in_tick invalid or unsupported (%u/%u)", me.TimeScale, me.NumUnitsInTick)
			me.TimingInfoPresentFlag = 0
		}

		me.FixedFrameRateFlag = uint8(gb.ReadBits(1))
	}

	me.NalHrdParametersPresentFlag = uint8(gb.ReadBits(1))
	if me.NalHrdParametersPresentFlag != 0 {
		err := me.NalHrd.parse(gb)
		if err != nil {
			return err
		}
	}

	me.VclHrdParametersPresentFlag = uint8(gb.ReadBits(1))
	if me.VclHrdParametersPresentFlag != 0 {
		err := me.VclHrd.parse(gb)
		if err != nil {
			return err
		}
	}

	if me.NalHrdParametersPresentFlag != 0 || me.VclHrdParametersPresentFlag != 0 {
		me.LowDelayHrdFlag = uint8(gb.ReadBits(1))
	}

	me.PicStructPresentFlag = uint8(gb.ReadBits(1))
	if gb.Left() == 0 {
		return nil
	}

	me.BitstreamRestrictionFlag = uint8(gb.ReadBits(1))
	if me.BitstreamRestrictionFlag != 0 {
		me.MotionVectorsOverPicBoundariesFlag = uint8(gb.ReadBits(1))
		me.MaxBytesPerPicDenom = uint32(gb.ReadUE())
		me.MaxBitsPerMbDenom = uint32(gb.ReadUE())
		me.Log2MaxMvLengthHorizontal = uint32(gb.ReadUE())
		me.Log2MaxMvLengthVertical = uint32(gb.ReadUE())
		me.MaxNumReorderFrames = uint32(gb.ReadUE())
		me.MaxDecFrameBuffering = uint32(gb.ReadUE())

		if gb.Left() < 0 {
			me.MaxNumReorderFrames = 0
			me.BitstreamRestrictionFlag = 0
		}

		if me.MaxNumReorderFrames > 16 /* max_dec_frame_buffering || max_dec_frame_buffering > 16 */ {
			me.MaxNumReorderFrames = 16
			return fmt.Errorf("clipping illegal MaxNumReorderFrames %d", me.MaxNumReorderFrames)
		}
	}

	return nil
}

// HRD (Hypothetical Reference Decoder)
type HRD struct {
	logger log.ILogger

	CpbCnt                       uint32     // cpb_cnt_minus1 + 1, see H.264 E.1.2
	BitRateScale                 uint8      // 4 bits
	CpbSizeScale                 uint8      // 4 bits
	BitRateValue                 [32]uint32 // bit_rate_value_minus1 + 1
	CpbSizeValue                 [32]uint32 // cpb_size_value_minus1 + 1
	CbrFlag                      uint32
	InitialCpbRemovalDelayLength uint32 // initial_cpb_removal_delay_length_minus1 + 1
	CpbRemovalDelayLength        uint32 // cpb_removal_delay_length_minus1 + 1
	DpbOutputDelayLength         uint32 // dpb_output_delay_length_minus1 + 1
	TimeOffsetLength             uint32
}

// Init this class.
func (me *HRD) Init(logger log.ILogger) *HRD {
	me.logger = logger
	return me
}

func (me *HRD) parse(gb *utils.Golomb) error {
	me.CpbCnt = uint32(gb.ReadUE()) + 1
	if me.CpbCnt > 32 {
		return fmt.Errorf("invalid me.CpbCnt %d", me.CpbCnt)
	}

	me.BitRateScale = uint8(gb.ReadBits(4))
	me.CpbSizeScale = uint8(gb.ReadBits(4))

	for i := uint32(0); i < me.CpbCnt; i++ {
		me.BitRateValue[i] = uint32(gb.ReadUE())
		me.CpbSizeValue[i] = uint32(gb.ReadUE())
		me.CbrFlag |= uint32(gb.ReadBits(1)) << i
	}

	me.InitialCpbRemovalDelayLength = uint32(gb.ReadBits(5)) + 1
	me.CpbRemovalDelayLength = uint32(gb.ReadBits(5)) + 1
	me.DpbOutputDelayLength = uint32(gb.ReadBits(5)) + 1
	me.TimeOffsetLength = uint32(gb.ReadBits(5))

	return nil
}
