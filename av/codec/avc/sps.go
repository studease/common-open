package avc

import (
	"github.com/studease/common/av"
	"github.com/studease/common/av/utils"
)

// SPS (Sequence Parameter Set)
type SPS struct {
	Golomb                         utils.Golomb
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

// VUI (Video Usability Information)
type VUI struct {
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

// HRD (Hypothetical Reference Decoder)
type HRD struct {
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
